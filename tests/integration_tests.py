#!/usr/bin/env python3
"""
Nuclear AO3 Integration Tests
Catches common issues before manual testing
"""

import requests
import json
import time
import subprocess
import psycopg2
from typing import Dict, List, Tuple
import sys
import os

class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'

class TestRunner:
    def __init__(self):
        self.base_url = "http://localhost:8080"
        self.services = {
            "api-gateway": "http://localhost:8080",
            "auth-service": "http://localhost:8081", 
            "work-service": "http://localhost:8082",
            "tag-service": "http://localhost:8083",
            "search-service": "http://localhost:8084"
        }
        self.test_results = []
        self.jwt_token = None
        
    def log(self, message: str, color: str = Colors.BLUE):
        print(f"{color}{message}{Colors.ENDC}")
        
    def success(self, message: str):
        self.log(f"✅ {message}", Colors.GREEN)
        
    def failure(self, message: str):
        self.log(f"❌ {message}", Colors.RED)
        
    def warning(self, message: str):
        self.log(f"⚠️  {message}", Colors.YELLOW)

    def test_service_health(self) -> bool:
        """Test that all services are running and healthy"""
        self.log("\n🔍 Testing Service Health...")
        
        all_healthy = True
        for service_name, url in self.services.items():
            try:
                response = requests.get(f"{url}/health", timeout=5)
                if response.status_code == 200:
                    self.success(f"{service_name} is healthy")
                else:
                    self.failure(f"{service_name} returned {response.status_code}")
                    all_healthy = False
            except requests.exceptions.RequestException as e:
                self.failure(f"{service_name} is not reachable: {e}")
                all_healthy = False
                
        return all_healthy

    def test_database_schema(self) -> bool:
        """Test that database schema has required tables and columns"""
        self.log("\n🗄️  Testing Database Schema...")
        
        try:
            conn = psycopg2.connect(
                host="localhost",
                database="ao3_nuclear", 
                user="ao3_user",
                password="ao3_password"
            )
            cur = conn.cursor()
            
            # Test critical tables exist
            required_tables = [
                "users", "works", "chapters", "tags", "creatorship", 
                "series", "collections", "bookmarks", "comments"
            ]
            
            schema_ok = True
            for table in required_tables:
                cur.execute(f"""
                    SELECT EXISTS (
                        SELECT FROM information_schema.tables 
                        WHERE table_name = '{table}'
                    );
                """)
                exists = cur.fetchone()[0]
                if exists:
                    self.success(f"Table '{table}' exists")
                else:
                    self.failure(f"Table '{table}' is missing")
                    schema_ok = False
            
            # Test critical columns in works table (the ones that caused issues)
            cur.execute("""
                SELECT column_name FROM information_schema.columns 
                WHERE table_name = 'works'
            """)
            works_columns = [row[0] for row in cur.fetchall()]
            
            required_works_columns = [
                "id", "title", "summary", "notes", "language", "rating",
                "category", "warnings", "fandoms", "characters", "relationships",
                "freeform_tags", "series_id", "max_chapters", "chapter_count",
                "created_at", "updated_at"
            ]
            
            for column in required_works_columns:
                if column in works_columns:
                    self.success(f"Works table has '{column}' column")
                else:
                    self.failure(f"Works table missing '{column}' column") 
                    schema_ok = False
                    
            conn.close()
            return schema_ok
            
        except Exception as e:
            self.failure(f"Database connection failed: {e}")
            return False

    def test_authentication_flow(self) -> bool:
        """Test complete authentication flow"""
        self.log("\n🔐 Testing Authentication Flow...")
        
        # Test user registration
        try:
            register_data = {
                "username": f"testuser_{int(time.time())}",
                "email": f"test_{int(time.time())}@example.com", 
                "password": "testpass123"
            }
            
            response = requests.post(
                f"{self.base_url}/api/v1/auth/register",
                json=register_data,
                timeout=10
            )
            
            if response.status_code == 201:
                self.success("User registration works")
            else:
                self.warning(f"Registration returned {response.status_code}: {response.text}")
                
        except Exception as e:
            self.failure(f"Registration failed: {e}")
            
        # Test login with known user
        try:
            login_data = {
                "username": "test",
                "password": "password"
            }
            
            response = requests.post(
                f"{self.base_url}/api/v1/auth/login",
                json=login_data,
                timeout=10
            )
            
            if response.status_code == 200:
                data = response.json()
                if "access_token" in data:
                    self.jwt_token = data["access_token"]
                    self.success("Login successful, JWT token received")
                    return True
                else:
                    self.failure("Login successful but no access_token in response")
            else:
                self.failure(f"Login failed: {response.status_code} - {response.text}")
                
        except Exception as e:
            self.failure(f"Login request failed: {e}")
            
        return False

    def test_tag_search_endpoints(self) -> bool:
        """Test tag search endpoints that were returning 400 errors"""
        self.log("\n🏷️  Testing Tag Search Endpoints...")
        
        tag_types = ["fandom", "character", "relationship", "freeform"]
        all_working = True
        
        for tag_type in tag_types:
            try:
                response = requests.get(
                    f"{self.base_url}/api/v1/tags/search",
                    params={"q": "test", "type": tag_type, "limit": 5},
                    timeout=10
                )
                
                if response.status_code == 200:
                    self.success(f"Tag search for '{tag_type}' works")
                else:
                    self.failure(f"Tag search for '{tag_type}' returned {response.status_code}")
                    all_working = False
                    
            except Exception as e:
                self.failure(f"Tag search for '{tag_type}' failed: {e}")
                all_working = False
                
        return all_working

    def test_work_creation_flow(self) -> bool:
        """Test complete work creation flow"""
        self.log("\n📝 Testing Work Creation Flow...")
        
        if not self.jwt_token:
            self.failure("No JWT token available for work creation test")
            return False
            
        # Test /my/works endpoint (was returning 404)
        try:
            response = requests.get(
                f"{self.base_url}/api/v1/my/works",
                headers={"Authorization": f"Bearer {self.jwt_token}"},
                timeout=10
            )
            
            if response.status_code == 200:
                self.success("/my/works endpoint works")
            else:
                self.failure(f"/my/works returned {response.status_code}")
                return False
                
        except Exception as e:
            self.failure(f"/my/works request failed: {e}")
            return False
            
        # Test work creation
        try:
            work_data = {
                "title": f"Test Work {int(time.time())}",
                "summary": "Test summary",
                "language": "en",
                "rating": "General Audiences",
                "category": ["Gen"],
                "warnings": ["No Archive Warnings Apply"],
                "fandoms": ["Test Fandom"],
                "characters": [],
                "relationships": [],
                "freeform_tags": ["Test Tag"]
            }
            
            response = requests.post(
                f"{self.base_url}/api/v1/works/",
                json=work_data,
                headers={
                    "Authorization": f"Bearer {self.jwt_token}",
                    "Content-Type": "application/json"
                },
                timeout=10
            )
            
            if response.status_code == 201:
                self.success("Work creation successful")
                return True
            elif response.status_code == 401:
                self.failure("Work creation failed: Authentication issue")
            elif "series_id" in response.text:
                self.warning("Work creation failed: Database schema issue (series_id column missing)")
            else:
                self.failure(f"Work creation failed: {response.status_code} - {response.text}")
                
        except Exception as e:
            self.failure(f"Work creation request failed: {e}")
            
        return False

    def test_api_gateway_routing(self) -> bool:
        """Test that API Gateway properly routes to all services"""
        self.log("\n🌐 Testing API Gateway Routing...")
        
        routes_to_test = [
            ("/api/v1/works", "work-service"),
            ("/api/v1/tags/search", "tag-service"), 
            ("/api/v1/auth/login", "auth-service"),
            ("/api/v1/search/works", "search-service")
        ]
        
        all_routing = True
        for route, expected_service in routes_to_test:
            try:
                response = requests.get(
                    f"{self.base_url}{route}",
                    timeout=5,
                    allow_redirects=False
                )
                
                # Check response headers for proxy service indicator
                proxy_service = response.headers.get('X-Proxy-Service', 'unknown')
                
                if response.status_code < 500:  # Not a gateway error
                    self.success(f"Route {route} → {proxy_service}")
                else:
                    self.failure(f"Route {route} returned {response.status_code}")
                    all_routing = False
                    
            except Exception as e:
                self.failure(f"Route {route} failed: {e}")
                all_routing = False
                
        return all_routing

    def test_docker_containers(self) -> bool:
        """Test that all Docker containers are running"""
        self.log("\n🐳 Testing Docker Containers...")
        
        try:
            result = subprocess.run(
                ["docker", "ps", "--format", "table {{.Names}}\t{{.Status}}"],
                capture_output=True, text=True, timeout=10
            )
            
            container_output = result.stdout
            required_containers = [
                "ao3_api_gateway", "ao3_work_service", "ao3_auth_service",
                "ao3_tag_service", "ao3_search_service", "ao3_postgres", "ao3_redis"
            ]
            
            all_running = True
            for container in required_containers:
                if container in container_output and "Up" in container_output:
                    self.success(f"Container {container} is running")
                else:
                    self.failure(f"Container {container} is not running")
                    all_running = False
                    
            return all_running
            
        except Exception as e:
            self.failure(f"Docker container check failed: {e}")
            return False

    def run_all_tests(self):
        """Run all tests and report results"""
        self.log(f"\n{Colors.BOLD}🧪 Nuclear AO3 Integration Test Suite{Colors.ENDC}")
        self.log("=" * 50)
        
        tests = [
            ("Docker Containers", self.test_docker_containers),
            ("Service Health", self.test_service_health),
            ("Database Schema", self.test_database_schema),
            ("API Gateway Routing", self.test_api_gateway_routing),
            ("Authentication Flow", self.test_authentication_flow),
            ("Tag Search Endpoints", self.test_tag_search_endpoints),
            ("Work Creation Flow", self.test_work_creation_flow),
        ]
        
        passed = 0
        total = len(tests)
        
        for test_name, test_func in tests:
            try:
                if test_func():
                    passed += 1
            except Exception as e:
                self.failure(f"{test_name} test crashed: {e}")
                
        # Summary
        self.log("\n" + "=" * 50)
        if passed == total:
            self.success(f"All {total} tests passed! 🎉")
        else:
            self.failure(f"{passed}/{total} tests passed")
            
        self.log(f"\n{Colors.BOLD}Test Summary:{Colors.ENDC}")
        self.log(f"• Passed: {passed}")
        self.log(f"• Failed: {total - passed}")
        self.log(f"• Total:  {total}")
        
        return passed == total

if __name__ == "__main__":
    runner = TestRunner()
    success = runner.run_all_tests()
    sys.exit(0 if success else 1)