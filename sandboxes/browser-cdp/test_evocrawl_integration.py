#!/usr/bin/env python3
"""
EvoCrawl Browser API Integration Test Script
Tests OpenSandbox integration with EvoCrawl browser APIs
"""

import requests
import json
import time
import sys
from typing import Dict, Any, Optional

# Configuration
EVOCRAWL_API_URL = "http://localhost:3002"
EVOCRAWL_API_KEY = "ec-083fd4b43d874603a184e7525a36bae8"

# Test results tracking
test_results = []

def print_section(title: str):
    """Print a section header"""
    print("\n" + "=" * 80)
    print(f"{title}")
    print("=" * 80)

def print_command(command: str):
    """Print a command"""
    print(f"\nCommand: {command}")
    print("-" * 80)

def print_response(status: int, body: Any):
    """Print response"""
    print(f"\nResponse Status: {status}")
    print(f"Response Body: {json.dumps(body, indent=2) if body else 'None'}")
    print("-" * 80)

def record_test(name: str, status: str, details: str = ""):
    """Record a test result"""
    test_results.append({
        "name": name,
        "status": status,
        "details": details
    })
    status_symbol = "✅" if status == "PASS" else "❌" if status == "FAIL" else "⚠️"
    print(f"\n{status_symbol} {name}: {status}")
    if details:
        print(f"   Details: {details}")

def make_request(
    method: str,
    endpoint: str,
    data: Optional[Dict[str, Any]] = None,
    params: Optional[Dict[str, Any]] = None
) -> tuple[int, Any]:
    """Make an HTTP request to EvoCrawl API"""
    url = f"{EVOCRAWL_API_URL}{endpoint}"
    headers = {
        "Authorization": f"Bearer {EVOCRAWL_API_KEY}",
        "Content-Type": "application/json"
    }
    
    try:
        if method == "GET":
            response = requests.get(url, headers=headers, params=params, timeout=30)
        elif method == "POST":
            response = requests.post(url, headers=headers, json=data, timeout=30)
        elif method == "DELETE":
            response = requests.delete(url, headers=headers, timeout=30)
        else:
            return 0, {"error": f"Unsupported method: {method}"}
        
        try:
            body = response.json()
        except:
            body = response.text
        
        return response.status_code, body
    except Exception as e:
        return 0, {"error": str(e)}

def test_browser_create():
    """Test browser session creation"""
    print_section("Test 1: Browser Create (POST /v2/browser)")
    
    endpoint = "/v2/browser"
    data = {
        "ttl": 600,
        "activityTtl": 300,
        "streamWebView": True
    }
    
    print_command(f"POST {endpoint} with data: {json.dumps(data, indent=2)}")
    status, body = make_request("POST", endpoint, data)
    print_response(status, body)
    
    if status == 200 and body.get("success"):
        print(f"\n✅ Browser created successfully")
        print(f"   Session ID: {body.get('id')}")
        print(f"   CDP URL: {body.get('cdpUrl')}")
        print(f"   Expires At: {body.get('expiresAt')}")
        record_test("evocrawl_browser_create", "PASS", f"Session ID: {body.get('id')}")
        return body.get('id')
    else:
        error_msg = body.get("error", "Unknown error") if isinstance(body, dict) else str(body)
        print(f"\n❌ Failed to create browser")
        print(f"   Error: {error_msg}")
        record_test("evocrawl_browser_create", "FAIL", error_msg)
        return None

def test_browser_execute(session_id: str):
    """Test code execution in browser session"""
    print_section("Test 2: Browser Execute (POST /v2/browser/:sessionId/exec)")
    
    if not session_id:
        print("⚠️ Skipping - no valid session ID")
        record_test("evocrawl_browser_execute", "SKIP", "No valid session ID")
        return
    
    endpoint = f"/v2/browser/{session_id}/execute"
    data = {
        "code": "echo 'Hello from OpenSandbox!'",
        "language": "bash",
        "timeout": 30,
        "responseFormat": "json"
    }
    
    print_command(f"POST {endpoint} with data: {json.dumps(data, indent=2)}")
    status, body = make_request("POST", endpoint, data)
    print_response(status, body)
    
    if status == 200 and body.get("success"):
        print(f"\n✅ Code executed successfully")
        print(f"   Exit Code: {body.get('exitCode')}")
        print(f"   Stdout: {body.get('stdout', '')[:100]}...")
        record_test("evocrawl_browser_execute", "PASS", f"Exit code: {body.get('exitCode')}")
    else:
        error_msg = body.get("error", "Unknown error") if isinstance(body, dict) else str(body)
        print(f"\n❌ Failed to execute code")
        print(f"   Error: {error_msg}")
        record_test("evocrawl_browser_execute", "FAIL", error_msg)

def test_browser_list():
    """Test listing browser sessions"""
    print_section("Test 3: Browser List (GET /v2/browser)")
    
    endpoint = "/v2/browser"
    
    print_command(f"GET {endpoint}")
    status, body = make_request("GET", endpoint)
    print_response(status, body)
    
    if status == 200 and body.get("success"):
        sessions = body.get("sessions", [])
        print(f"\n✅ Browser list retrieved successfully")
        print(f"   Total sessions: {len(sessions)}")
        for session in sessions[:3]:  # Show first 3
            print(f"   - ID: {session.get('id')}, Status: {session.get('status')}")
        record_test("evocrawl_browser_list", "PASS", f"Found {len(sessions)} sessions")
    else:
        error_msg = body.get("error", "Unknown error") if isinstance(body, dict) else str(body)
        print(f"\n❌ Failed to list browsers")
        print(f"   Error: {error_msg}")
        record_test("evocrawl_browser_list", "FAIL", error_msg)

def test_browser_delete(session_id: str):
    """Test browser session deletion"""
    print_section("Test 4: Browser Delete (DELETE /v2/browser/:sessionId)")
    
    if not session_id:
        print("⚠️ Skipping - no valid session ID")
        record_test("evocrawl_browser_delete", "SKIP", "No valid session ID")
        return
    
    endpoint = f"/v2/browser/{session_id}"
    
    print_command(f"DELETE {endpoint}")
    status, body = make_request("DELETE", endpoint)
    print_response(status, body)
    
    if status == 200 and body.get("success"):
        print(f"\n✅ Browser deleted successfully")
        record_test("evocrawl_browser_delete", "PASS", f"Deleted session: {session_id}")
    else:
        error_msg = body.get("error", "Unknown error") if isinstance(body, dict) else str(body)
        print(f"\n❌ Failed to delete browser")
        print(f"   Error: {error_msg}")
        record_test("evocrawl_browser_delete", "FAIL", error_msg)

def test_extract():
    """Test extract endpoint"""
    print_section("Test 5: Extract (POST /v1/extract)")
    
    endpoint = "/v1/extract"
    data = {
        "urls": ["https://example.com"],
        "prompt": "Extract the main heading from the page"
    }
    
    print_command(f"POST {endpoint} with data: {json.dumps(data, indent=2)}")
    status, body = make_request("POST", endpoint, data)
    print_response(status, body)
    
    # Extract jobs are expected to fail without proper setup
    if status == 200:
        print(f"\n✅ Extract job created")
        record_test("evocrawl_extract", "PASS", f"Job ID: {body.get('id', 'N/A')}")
    elif status == 400 or status == 401:
        error_msg = body.get("error", "Unknown error") if isinstance(body, dict) else str(body)
        print(f"\n⚠️ Extract failed (expected - needs job setup)")
        print(f"   Error: {error_msg}")
        record_test("evocrawl_extract", "FAIL", f"Expected failure: {error_msg}")
    else:
        error_msg = body.get("error", "Unknown error") if isinstance(body, dict) else str(body)
        print(f"\n❌ Extract failed unexpectedly")
        print(f"   Error: {error_msg}")
        record_test("evocrawl_extract", "FAIL", error_msg)

def test_agent():
    """Test agent endpoint (v2)"""
    print_section("Test 6: Agent (POST /v2/agent)")
    
    endpoint = "/v2/agent"
    data = {
        "prompt": "Find information about AI startups"
    }
    
    print_command(f"POST {endpoint} with data: {json.dumps(data, indent=2)}")
    status, body = make_request("POST", endpoint, data)
    print_response(status, body)
    
    # Agent is expected to fail without proper service configuration
    if status == 200:
        print(f"\n✅ Agent job created")
        record_test("evocrawl_agent", "PASS", f"Job ID: {body.get('id', 'N/A')}")
    elif status == 400 or status == 503:
        error_msg = body.get("error", "Unknown error") if isinstance(body, dict) else str(body)
        print(f"\n⚠️ Agent failed (expected - needs service configuration)")
        print(f"   Error: {error_msg}")
        record_test("evocrawl_agent", "FAIL", f"Expected failure: {error_msg}")
    else:
        error_msg = body.get("error", "Unknown error") if isinstance(body, dict) else str(body)
        print(f"\n❌ Agent failed unexpectedly")
        print(f"   Error: {error_msg}")
        record_test("evocrawl_agent", "FAIL", error_msg)

def test_interact():
    """Test interact endpoint (v2)"""
    print_section("Test 7: Interact (POST /v2/scrape/:jobId/interact)")
    
    # Note: This requires a valid scrape job ID from a previous extract
    # For now, we'll skip this test as it needs a real scrape job
    print("⚠️ Skipping - requires valid scrape job ID from extract endpoint")
    record_test("evocrawl_interact", "SKIP", "Requires valid scrape job ID")
    return

def generate_report():
    """Generate test report"""
    print_section("Test Report Summary")
    
    total = len(test_results)
    passed = sum(1 for r in test_results if r["status"] == "PASS")
    failed = sum(1 for r in test_results if r["status"] == "FAIL")
    skipped = sum(1 for r in test_results if r["status"] == "SKIP")
    
    print(f"\nTotal Tests: {total}")
    print(f"✅ Passed: {passed}")
    print(f"❌ Failed: {failed}")
    print(f"⚠️ Skipped: {skipped}")
    
    print("\n" + "-" * 80)
    print("Detailed Results:")
    print("-" * 80)
    
    for result in test_results:
        status_symbol = "✅" if result["status"] == "PASS" else "❌" if result["status"] == "FAIL" else "⚠️"
        print(f"\n{status_symbol} {result['name']}: {result['status']}")
        if result["details"]:
            print(f"   {result['details']}")
    
    print("\n" + "=" * 80)
    
    # Save report to file
    report_file = "/tmp/evocrawl_integration_test_report.json"
    with open(report_file, "w") as f:
        json.dump({
            "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
            "summary": {
                "total": total,
                "passed": passed,
                "failed": failed,
                "skipped": skipped
            },
            "results": test_results
        }, f, indent=2)
    
    print(f"\n📄 Report saved to: {report_file}")
    print("=" * 80)

def main():
    """Main test execution"""
    print_section("EvoCrawl Browser API Integration Test")
    print(f"\nAPI URL: {EVOCRAWL_API_URL}")
    print(f"API Key: {EVOCRAWL_API_KEY[:20]}...")
    print(f"Timestamp: {time.strftime('%Y-%m-%d %H:%M:%S')}")
    
    session_id = None
    
    try:
        # Test browser create
        session_id = test_browser_create()
        
        # Wait a bit for session to be ready
        if session_id:
            time.sleep(2)
        
        # Test browser execute
        test_browser_execute(session_id)
        
        # Test browser list
        test_browser_list()
        
        # Test browser delete
        test_browser_delete(session_id)
        
        # Test other endpoints (expected to fail without proper setup)
        test_extract()
        test_agent()
        test_interact()
        
    except KeyboardInterrupt:
        print("\n\n⚠️ Test interrupted by user")
    except Exception as e:
        print(f"\n\n❌ Test execution failed with error: {e}")
        import traceback
        traceback.print_exc()
    finally:
        # Generate report
        generate_report()
        
        # Cleanup if session still exists
        if session_id:
            print("\n🧹 Cleaning up session if needed...")
            try:
                make_request("DELETE", f"/v2/browser/{session_id}")
            except:
                pass

if __name__ == "__main__":
    main()
