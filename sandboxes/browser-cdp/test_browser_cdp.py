#!/usr/bin/env python3
"""
Browser CDP Sandbox Test Script

This script automates the 4-step process to test the browser-cdp sandbox:
1. Create sandbox
2. Get execd endpoint
3. Call execd to start browser
4. Construct CDP URL

Usage:
    python test_browser_cdp.py
"""

import requests
import json
import sys

# Configuration
API_KEY = "111-111-222-aaa-aaa"
API_BASE = "https://sandbox.pazity.com/v1"
SANDBOX_IMAGE = "opensandbox/browser-cdp:latest"
EXECD_PORT = 44772


def print_step(step_num, description, command, response=None, error=None):
    """Print step details with command and result"""
    print(f"\n{'='*80}")
    print(f"Step {step_num}: {description}")
    print(f"{'='*80}")
    print(f"Command: {command}")
    print(f"{'-'*80}")
    if response:
        print(f"Response Status: {response.status_code}")
        try:
            print(f"Response Body: {json.dumps(response.json(), indent=2)}")
        except:
            print(f"Response Body: {response.text}")
    if error:
        print(f"Error: {error}")
    print(f"{'='*80}\n")


def step1_create_sandbox():
    """Step 1: Create sandbox"""
    url = f"{API_BASE}/sandboxes"
    headers = {
        "OPEN-SANDBOX-API-KEY": API_KEY,
        "Content-Type": "application/json"
    }
    data = {
        "image": {
            "uri": SANDBOX_IMAGE
        },
        "entrypoint": ["/opt/opensandbox/browser-entrypoint.sh"],
        "timeout": 600,
        "resourceLimits": {
            "cpu": "300m",
            "memory": "1Gi"
        }
    }

    command = f"curl -X POST '{url}' -H 'OPEN-SANDBOX-API-KEY: {API_KEY}' -H 'Content-Type: application/json' -d '{json.dumps(data)}'"

    try:
        response = requests.post(url, headers=headers, json=data, timeout=30)
        print_step(1, "Create Sandbox", command, response)

        if response.status_code in [200, 201, 202]:
            # Response uses "id" not "sandboxId"
            return response.json().get("id")
        else:
            print(f"Failed to create sandbox. Status: {response.status_code}")
            try:
                print(f"Error Response: {response.text}")
            except:
                pass
            return None
    except Exception as e:
        print_step(1, "Create Sandbox", command, error=str(e))
        return None


def step2_get_endpoint(sandbox_id):
    """Step 2: Get execd endpoint"""
    url = f"{API_BASE}/sandboxes/{sandbox_id}/endpoints/{EXECD_PORT}?use_server_proxy=true"
    headers = {
        "OPEN-SANDBOX-API-KEY": API_KEY
    }

    command = f"curl -X GET '{url}' -H 'OPEN-SANDBOX-API-KEY: {API_KEY}'"

    try:
        response = requests.get(url, headers=headers, timeout=30)
        print_step(2, "Get Execd Endpoint", command, response)

        if response.status_code == 200:
            return response.json().get("endpoint")
        else:
            print(f"Failed to get endpoint. Status: {response.status_code}")
            try:
                print(f"Error Response: {response.text}")
            except:
                pass
            return None
    except Exception as e:
        print_step(2, "Get Execd Endpoint", command, error=str(e))
        return None


def step3_start_browser(endpoint):
    """Step 3: Call execd to start browser"""
    # Add https:// scheme if not present
    if not endpoint.startswith("http://") and not endpoint.startswith("https://"):
        endpoint = f"https://{endpoint}"
    
    url = f"{endpoint}/browser"
    headers = {
        "Content-Type": "application/json",
        "OPEN-SANDBOX-API-KEY": API_KEY
    }

    command = f"curl -X POST '{url}' -H 'Content-Type: application/json' -H 'OPEN-SANDBOX-API-KEY: {API_KEY}'"

    try:
        response = requests.post(url, headers=headers, timeout=30)
        print_step(3, "Start Browser", command, response)

        if response.status_code == 200:
            return response.json()
        else:
            print(f"Failed to start browser. Status: {response.status_code}")
            try:
                print(f"Error Response: {response.text}")
            except:
                pass
            return None
    except Exception as e:
        print_step(3, "Start Browser", command, error=str(e))
        return None


def step4_construct_cdp_url(sandbox_id, browser_response):
    """Step 4: Construct CDP URL"""
    if not browser_response:
        print("Cannot construct CDP URL: browser response is None")
        return None

    cdp_port = browser_response.get("cdpPort")
    if not cdp_port:
        print("Cannot construct CDP URL: cdpPort not found in response")
        return None

    cdp_url = f"ws://sandbox.pazity.com/sandboxes/{sandbox_id}/proxy/{cdp_port}"
    
    command = f"curl -i -N -H 'Connection: Upgrade' -H 'Upgrade: websocket' -H 'Sec-WebSocket-Version: 13' -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' '{cdp_url}'"

    print_step(4, "Construct CDP URL", command)
    print(f"CDP URL: {cdp_url}")
    print(f"Note: This is a WebSocket URL. Use a WebSocket client to connect.")
    
    return cdp_url


def main():
    """Main execution"""
    print("Browser CDP Sandbox Test Script")
    print("="*80)

    # Step 1: Create sandbox
    sandbox_id = step1_create_sandbox()
    if not sandbox_id:
        print("\n❌ Step 1 failed. Cannot proceed.")
        sys.exit(1)

    print(f"✅ Sandbox created with ID: {sandbox_id}")

    # Step 2: Get endpoint
    endpoint = step2_get_endpoint(sandbox_id)
    if not endpoint:
        print("\n❌ Step 2 failed. Cannot proceed.")
        sys.exit(1)

    print(f"✅ Endpoint obtained: {endpoint}")

    # Step 3: Start browser
    browser_response = step3_start_browser(endpoint)
    if not browser_response:
        print("\n❌ Step 3 failed. Cannot proceed.")
        sys.exit(1)

    print(f"✅ Browser started")

    # Step 4: Construct CDP URL
    cdp_url = step4_construct_cdp_url(sandbox_id, browser_response)
    if not cdp_url:
        print("\n❌ Step 4 failed.")
        sys.exit(1)

    print(f"✅ CDP URL constructed successfully")
    print(f"\n{'='*80}")
    print("Test completed successfully!")
    print(f"{'='*80}\n")


if __name__ == "__main__":
    main()
