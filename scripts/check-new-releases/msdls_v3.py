import argparse
import json
import logging
import uuid
import time
import requests
import sys

# New v3 JSON Endpoints
BASE_URL = "https://www.microsoft.com/software-download-connector/api"
PROFILE = "606624d44113"
LOCALE = "en-US"
UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

# Extended ranges for Copilot OS / Project Aion discovery
COPILOT_OS_RANGES = [
    (29000, 31000),   # Main Copilot+ PC / Project Aion range
    (28000, 29000),   # Pre-release range
    (31000, 32000),   # Future range
]

def setup_session():
    """Initializes a session with Microsoft tracking servers."""
    session_id = str(uuid.uuid4())
    try:
        requests.get(
            "https://vlscppe.microsoft.com/tags",
            params={"org_id": "y6jn8c31", "session_id": session_id},
            headers={"User-Agent": UA},
            timeout=10,
        )
    except Exception as e:
        logging.warning(f"Session permit call failed (continuing anyway): {e}")
    return session_id

def get_product(product_id, session_id):
    """Fetches product/SKU info for a given ID."""
    url = f"{BASE_URL}/getskuinformationbyproductedition"
    params = {
        "profile": PROFILE,
        "productEditionId": product_id,
        "SKU": "undefined",
        "friendlyFileName": "undefined",
        "Locale": LOCALE,
        "sessionID": session_id
    }
    headers = {
        "User-Agent": UA, 
        "Accept": "application/json",
        "Referer": "https://www.microsoft.com/en-us/software-download/windows11"
    }
    
    try:
        r = requests.get(url, params=params, headers=headers, timeout=15)
        if not r.ok: 
            return None
        
        # Handle Microsoft's double-encoded JSON
        raw_text = r.text
        try:
            data = json.loads(raw_text)
            if isinstance(data, str):
                data = json.loads(data)
            return data
        except:
            return None
    except Exception as e:
        logging.error(f"Error fetching ID {product_id}: {e}")
        return None

def scan_id(product_id):
    """Checks if a product ID is valid and active."""
    session_id = setup_session()
    data = get_product(product_id, session_id)
    
    if not data:
        return None
    
    if "Errors" in data and data["Errors"] and len(data["Errors"]) > 0:
        return None

    # MS usually puts the release name in the first SKU
    skus = data.get("Skus", [])
    if not skus:
        return None
        
    s = skus[0]
    name = s.get("ProductDisplayName") or s.get("LocalizedProductDisplayName")
    if not name:
        return None
        
    # Tag Copilot+ PC / Project Aion builds
    try:
        pid_int = int(product_id)
        if pid_int >= 28000:
            return f"{name} (Copilot+ PC / Project Aion)"
    except ValueError:
        pass
    return name

def scan_range(first, last, label=""):
    """Scan a range of product IDs and return found products."""
    logging.info(f"Scanning range {first} to {last} {label}...")
    products = {}
    
    for i in range(first, last + 1):
        name = scan_id(i)
        if name:
            logging.info(f"  FOUND: [{i}] {name}")
            products[str(i)] = {
                "name": name,
                "badge": "Copilot+ PC" if i >= 28000 else "",
                "archs": [],
                "related": [],
                "active": True
            }
        
        # Be gentle to the API
        time.sleep(0.3)
    
    return products

def scan_copilot_os_ranges():
    """Scan all known Copilot OS / Project Aion ranges."""
    all_products = {}
    for start, end in COPILOT_OS_RANGES:
        results = scan_range(start, end, f"(Copilot OS range)")
        all_products.update(results)
    return all_products

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Scan for new Windows Product IDs (v3 JSON API)")
    parser.add_argument("--first", type=int, help="First ID to check")
    parser.add_argument("--last", type=int, help="Last ID to check")
    parser.add_argument("--copilot", action="store_true", help="Scan Copilot OS / Project Aion ranges")
    parser.add_argument("--write", help="Output JSON file")
    parser.add_argument("--all", action="store_true", help="Scan all ranges including Copilot OS")
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO, format='%(message)s')
    
    products = {}
    
    if args.all or args.copilot:
        products.update(scan_copilot_os_ranges())
    
    if args.first is not None and args.last is not None:
        products.update(scan_range(args.first, args.last))
    
    if not products:
        logging.info("No products found in scanned ranges.")
    
    if args.write:
        import os
        catalog = {}
        if os.path.exists(args.write):
            try:
                with open(args.write, 'r', encoding='utf-8') as f:
                    catalog = json.load(f)
            except Exception as e:
                logging.error(f"Failed to read existing catalog: {e}")
        
        # Merge new products
        for pid, info in products.items():
            if pid in catalog:
                if isinstance(catalog[pid], dict):
                    catalog[pid].update(info)
                    catalog[pid]["active"] = True
                else:
                    catalog[pid] = info
            else:
                catalog[pid] = info
                
        with open(args.write, 'w', encoding='utf-8') as f:
            json.dump(catalog, f, indent=4)
            logging.info(f"\nSaved {len(catalog)} products to {args.write}")
    
    logging.info("Done.")
