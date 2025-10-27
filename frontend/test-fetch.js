// Test script to verify API connectivity from Node.js environment
const fetch = require('node-fetch');

async function testAPI() {
    console.log('Testing API connectivity...');
    
    try {
        console.log('Making request to http://localhost:8082/api/v1/works');
        const response = await fetch('http://localhost:8082/api/v1/works');
        
        console.log('Response status:', response.status);
        console.log('Response headers:', response.headers.raw());
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        console.log('✅ Success! Data received:');
        console.log('Works count:', data.works ? data.works.length : 0);
        console.log('First work:', data.works?.[0] ? data.works[0].title : 'None');
        
        return data;
    } catch (error) {
        console.error('❌ Error:', error.message);
        throw error;
    }
}

testAPI()
    .then(() => console.log('Test completed successfully'))
    .catch(() => console.log('Test failed'));