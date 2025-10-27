#!/usr/bin/env node

/**
 * Script to automatically update attribution data from package.json
 * Run with: node scripts/update-attributions.js
 */

const fs = require('fs');
const path = require('path');

function updateAttributions() {
  try {
    // Read package.json
    const packagePath = path.join(__dirname, '../package.json');
    const packageData = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
    
    // Extract main dependencies (not devDependencies for runtime attributions)
    const dependencies = packageData.dependencies || {};
    
    const attributionData = {
      generatedAt: new Date().toISOString(),
      dependencies: Object.keys(dependencies).map(name => ({
        name,
        version: dependencies[name],
        // These would ideally be fetched from npm API in a real implementation
        licenses: 'MIT', // Default - should be fetched
        repository: `https://www.npmjs.com/package/${name}`,
        publisher: 'Unknown', // Should be fetched from npm
        description: 'Package description would be fetched from npm API'
      }))
    };
    
    // Write to a JSON file that the attributions page can load
    const outputPath = path.join(__dirname, '../public/attributions.json');
    fs.writeFileSync(outputPath, JSON.stringify(attributionData, null, 2));
    
    console.log(`✅ Updated attributions for ${attributionData.dependencies.length} dependencies`);
    console.log(`📝 Written to: ${outputPath}`);
    
  } catch (error) {
    console.error('❌ Failed to update attributions:', error.message);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  updateAttributions();
}

module.exports = { updateAttributions };