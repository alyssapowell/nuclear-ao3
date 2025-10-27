#!/usr/bin/env python3
"""
Comprehensive Nuclear AO3 Dashboard API Test Suite

Tests all dashboard and authentication functionality that was implemented.
"""

try:
    import requests
except ImportError:
    print("Installing requests library...")
    import subprocess
    import sys
    subprocess.check_call([sys.executable, "-m", "pip", "install", "requests"])
    import requests

import json
import time
import sys
import os
from typing import Dict, Optional

class DashboardAPITester:
    def __init__(self):
        self.base_url = "http://localhost:8080"
        self.auth_url = "http://localhost:8081"
        self.work_url = "http://localhost:8082"
        self.test_user = {
            "email": "testuser30d_v2@example.com",
            "password": "TestPassword123!"
        }
        self.access_token = None
        self.test_work_id = None
        self.passed_tests = 0
        self.failed_tests = 0
        
    def log(self, message: str, status: str = "INFO"):
        colors = {
            "INFO": "\033[0;34m",  # Blue
            "PASS": "\033[0;32m",  # Green  
            "FAIL": "\033[0;31m",  # Red
            "WARN": "\033[0;33m",  # Yellow
            "RESET": "\033[0m"     # Reset
        }
        print(f"{colors.get(status, '')}{status}: {message}{colors['RESET']}")
        
    def test_assert(self, condition: bool, test_name: str, error_msg: str = ""):
        """Assert a condition and track test results"""
        if condition:
            self.log(f"✅ {test_name}", "PASS")
            self.passed_tests += 1
            return True
        else:
            self.log(f"❌ {test_name} - {error_msg}", "FAIL")
            self.failed_tests += 1
            return False
            
    def test_service_health(self) -> bool:
        """Test all service health endpoints"""
        self.log("Testing Service Health Endpoints", "INFO")
        
        services = [
            ("API Gateway", f"{self.base_url}/health"),
            ("Auth Service", f"{self.auth_url}/health"), 
            ("Work Service", f"{self.work_url}/health")
        ]
        
        all_healthy = True
        for name, url in services:
            try:
                response = requests.get(url, timeout=5)
                healthy = response.status_code == 200 and "healthy" in response.text
                all_healthy &= self.test_assert(
                    healthy, 
                    f"{name} Health Check",
                    f"Status: {response.status_code}, Response: {response.text[:100]}"
                )
            except Exception as e:
                all_healthy &= self.test_assert(False, f"{name} Health Check", str(e))
                
        return all_healthy
        
    def test_authentication_flow(self) -> bool:
        """Test complete authentication flow"""
        self.log("Testing Authentication Flow", "INFO")
        
        # Test login with valid credentials
        try:
            login_data = {
                "email": self.test_user["email"],
                "password": self.test_user["password"]
            }
            
            response = requests.post(
                f"{self.auth_url}/api/v1/auth/login",
                json=login_data,
                timeout=10
            )
            
            login_success = self.test_assert(
                response.status_code == 200,
                "User Login Success",
                f"Status: {response.status_code}, Response: {response.text[:200]}"
            )
            
            if login_success:
                data = response.json()
                self.access_token = data.get("access_token")
                
                # Validate token structure
                token_valid = self.test_assert(
                    self.access_token is not None and len(self.access_token) > 50,
                    "JWT Token Generation",
                    f"Token length: {len(self.access_token) if self.access_token else 0}"
                )
                
                # Test token validation via /me endpoint
                if token_valid:
                    headers = {"Authorization": f"Bearer {self.access_token}"}
                    me_response = requests.get(
                        f"{self.auth_url}/api/v1/auth/me",
                        headers=headers,
                        timeout=5
                    )
                    
                    me_success = self.test_assert(
                        me_response.status_code == 200,
                        "Token Validation (/me endpoint)",
                        f"Status: {me_response.status_code}"
                    )
                    
                    if me_success:
                        me_data = me_response.json()
                        user_id_extracted = self.test_assert(
                            "user_id" in me_data and len(me_data["user_id"]) == 36,
                            "User ID Extraction from Token",
                            f"User ID: {me_data.get('user_id', 'missing')}"
                        )
                        
                return login_success and token_valid
                        
        except Exception as e:
            self.test_assert(False, "Authentication Flow", str(e))
            return False
            
        return False
        
    def test_dashboard_api(self) -> bool:
        """Test dashboard API functionality"""
        self.log("Testing Dashboard API", "INFO")
        
        if not self.access_token:
            self.test_assert(False, "Dashboard API Prerequisites", "No access token available")
            return False
            
        headers = {"Authorization": f"Bearer {self.access_token}"}
        
        # Test GetMyWorks endpoint
        try:
            response = requests.get(
                f"{self.work_url}/api/v1/my/works",
                headers=headers,
                timeout=10
            )
            
            dashboard_success = self.test_assert(
                response.status_code == 200,
                "Dashboard API Response",
                f"Status: {response.status_code}, Response: {response.text[:200]}"
            )
            
            if dashboard_success:
                data = response.json()
                works_array = data.get("works", [])
                
                # Test response structure
                structure_valid = self.test_assert(
                    isinstance(works_array, list),
                    "Dashboard Response Structure",
                    f"Expected list, got {type(works_array)}"
                )
                
                # Test that user has works (based on our previous testing)
                has_works = self.test_assert(
                    len(works_array) > 0,
                    "User Has Works in Dashboard",
                    f"Found {len(works_array)} works"
                )
                
                if has_works and len(works_array) > 0:
                    work = works_array[0]
                    self.test_work_id = work.get("id")
                    
                    # Test work object structure
                    required_fields = ["id", "title", "status", "updated_at", "word_count"]
                    fields_present = all(field in work for field in required_fields)
                    
                    self.test_assert(
                        fields_present,
                        "Work Object Fields Complete",
                        f"Missing fields: {[f for f in required_fields if f not in work]}"
                    )
                    
                    # Test work data validity
                    work_id_valid = self.test_assert(
                        len(work.get("id", "")) == 36,
                        "Work ID Format Valid",
                        f"Work ID: {work.get('id', 'missing')}"
                    )
                    
                    title_valid = self.test_assert(
                        len(work.get("title", "")) > 0,
                        "Work Title Present",
                        f"Title: '{work.get('title', 'missing')}'"
                    )
                    
                    return dashboard_success and structure_valid and has_works and work_id_valid and title_valid
                    
                return dashboard_success and structure_valid
                
        except Exception as e:
            self.test_assert(False, "Dashboard API", str(e))
            return False
            
        return False
        
    def test_work_viewing(self) -> bool:
        """Test work viewing functionality"""
        self.log("Testing Work Viewing", "INFO")
        
        if not self.access_token or not self.test_work_id:
            self.test_assert(False, "Work Viewing Prerequisites", "No access token or work ID available")
            return False
            
        headers = {"Authorization": f"Bearer {self.access_token}"}
        
        # Test individual work viewing
        try:
            response = requests.get(
                f"{self.work_url}/api/v1/works/{self.test_work_id}",
                headers=headers,
                timeout=10
            )
            
            view_success = self.test_assert(
                response.status_code == 200,
                "Work Viewing API Response",
                f"Status: {response.status_code}, Response: {response.text[:200]}"
            )
            
            if view_success:
                data = response.json()
                work_data = data.get("work", {})
                
                # Test work data completeness
                work_complete = self.test_assert(
                    work_data.get("id") == self.test_work_id,
                    "Work ID Consistency",
                    f"Expected: {self.test_work_id}, Got: {work_data.get('id')}"
                )
                
                # Test author information
                authors = data.get("authors", [])
                author_present = self.test_assert(
                    len(authors) > 0,
                    "Work Author Information Present",
                    f"Found {len(authors)} authors"
                )
                
                if author_present:
                    author = authors[0]
                    author_valid = self.test_assert(
                        "user_id" in author and "username" in author,
                        "Author Data Structure Valid",
                        f"Author keys: {list(author.keys())}"
                    )
                    return view_success and work_complete and author_present and author_valid
                else:
                    return view_success and work_complete and author_present
                
        except Exception as e:
            self.test_assert(False, "Work Viewing", str(e))
            return False
            
        return False
        
    def test_permission_system(self) -> bool:
        """Test work permission system"""
        self.log("Testing Permission System", "INFO")
        
        if not self.test_work_id:
            self.test_assert(False, "Permission Testing Prerequisites", "No work ID available")
            return False
            
        # Test authenticated access (should work - user owns the work)
        headers = {"Authorization": f"Bearer {self.access_token}"}
        
        try:
            auth_response = requests.get(
                f"{self.work_url}/api/v1/works/{self.test_work_id}",
                headers=headers,
                timeout=5
            )
            
            auth_access = self.test_assert(
                auth_response.status_code == 200,
                "Authenticated Work Access",
                f"Status: {auth_response.status_code}"
            )
            
            # Test unauthenticated access (should fail for draft works)
            unauth_response = requests.get(
                f"{self.work_url}/api/v1/works/{self.test_work_id}",
                timeout=5
            )
            
            # Note: This might return 200 if work is published, or 403 if draft
            unauth_handled = self.test_assert(
                unauth_response.status_code in [200, 403],
                "Unauthenticated Access Handling",
                f"Status: {unauth_response.status_code} (expected 200 for published or 403 for draft)"
            )
            
            return auth_access and unauth_handled
            
        except Exception as e:
            self.test_assert(False, "Permission System", str(e))
            return False
            
    def test_api_gateway_routing(self) -> bool:
        """Test API Gateway routing functionality"""
        self.log("Testing API Gateway Routing", "INFO")
        
        if not self.access_token:
            self.test_assert(False, "Gateway Routing Prerequisites", "No access token available") 
            return False
            
        headers = {"Authorization": f"Bearer {self.access_token}"}
        
        # Test dashboard route through gateway
        try:
            gateway_response = requests.get(
                f"{self.base_url}/api/v1/my/works",
                headers=headers,
                timeout=10
            )
            
            gateway_success = self.test_assert(
                gateway_response.status_code == 200,
                "Gateway Dashboard Routing",
                f"Status: {gateway_response.status_code}"
            )
            
            if gateway_success:
                # Test that gateway response matches direct service response
                direct_response = requests.get(
                    f"{self.work_url}/api/v1/my/works",
                    headers=headers,
                    timeout=5
                )
                
                responses_match = self.test_assert(
                    gateway_response.json() == direct_response.json(),
                    "Gateway vs Direct Service Consistency",
                    "Response data mismatch between gateway and direct service"
                )
                
                return gateway_success and responses_match
                
        except Exception as e:
            self.test_assert(False, "API Gateway Routing", str(e))
            return False
            
        return False
        
    def test_error_handling(self) -> bool:
        """Test API error handling"""
        self.log("Testing Error Handling", "INFO")
        
        # Test invalid token
        try:
            invalid_headers = {"Authorization": "Bearer invalid_token_12345"}
            
            invalid_response = requests.get(
                f"{self.work_url}/api/v1/my/works",
                headers=invalid_headers,
                timeout=5
            )
            
            invalid_handled = self.test_assert(
                invalid_response.status_code == 401,
                "Invalid Token Rejection", 
                f"Status: {invalid_response.status_code}, expected 401"
            )
            
            # Test missing authorization
            no_auth_response = requests.get(
                f"{self.work_url}/api/v1/my/works",
                timeout=5
            )
            
            no_auth_handled = self.test_assert(
                no_auth_response.status_code == 401,
                "Missing Authorization Rejection",
                f"Status: {no_auth_response.status_code}, expected 401"
            )
            
            # Test nonexistent work
            if self.access_token:
                headers = {"Authorization": f"Bearer {self.access_token}"}
                fake_work_id = "00000000-0000-0000-0000-000000000000"
                
                fake_response = requests.get(
                    f"{self.work_url}/api/v1/works/{fake_work_id}",
                    headers=headers,
                    timeout=5
                )
                
                fake_handled = self.test_assert(
                    fake_response.status_code in [404, 403],
                    "Nonexistent Work Handling",
                    f"Status: {fake_response.status_code}, expected 404 or 403"
                )
                
                return invalid_handled and no_auth_handled and fake_handled
                
            return invalid_handled and no_auth_handled
            
        except Exception as e:
            self.test_assert(False, "Error Handling", str(e))
            return False
        
    def run_all_tests(self) -> Dict[str, bool]:
        """Run all test suites"""
        self.log("🧪 Starting Nuclear AO3 Dashboard API Test Suite", "INFO")
        self.log("=" * 60, "INFO")
        
        results = {}
        
        # Run test suites in order
        test_suites = [
            ("Service Health", self.test_service_health),
            ("Authentication Flow", self.test_authentication_flow),
            ("Dashboard API", self.test_dashboard_api),
            ("Work Viewing", self.test_work_viewing),
            ("Permission System", self.test_permission_system),
            ("API Gateway Routing", self.test_api_gateway_routing),
            ("Error Handling", self.test_error_handling)
        ]
        
        for suite_name, test_func in test_suites:
            self.log(f"\n📋 Running {suite_name} Tests...", "INFO")
            try:
                results[suite_name] = test_func()
            except Exception as e:
                self.log(f"Test suite {suite_name} crashed: {e}", "FAIL")
                results[suite_name] = False
        
        # Print final results
        self.log("\n" + "=" * 60, "INFO")
        self.log("🎯 Test Results Summary:", "INFO")
        
        total_suites = len(results)
        passed_suites = sum(1 for result in results.values() if result)
        
        for suite_name, passed in results.items():
            status = "PASS" if passed else "FAIL"
            self.log(f"  {suite_name}: {'✅ PASSED' if passed else '❌ FAILED'}", status)
            
        self.log(f"\n📊 Overall Results:", "INFO")
        self.log(f"  Test Suites: {passed_suites}/{total_suites} passed", "INFO")
        self.log(f"  Individual Tests: {self.passed_tests}/{self.passed_tests + self.failed_tests} passed", "INFO")
        
        overall_success = passed_suites == total_suites
        
        if overall_success:
            self.log("\n🎉 ALL TESTS PASSED! Dashboard API is robust and ready for production.", "PASS")
        else:
            self.log(f"\n⚠️  {total_suites - passed_suites} test suite(s) failed. Review failures above.", "FAIL")
            
        return results

def main():
    """Main test runner"""
    tester = DashboardAPITester()
    results = tester.run_all_tests()
    
    # Exit with appropriate code
    all_passed = all(results.values())
    sys.exit(0 if all_passed else 1)

if __name__ == "__main__":
    main()