# UCRP Metadata Embedding Standards
## Universal Creator Rights Protocol - Content Tagging

### 🎯 **Overview**

This document defines standardized methods for embedding Universal Creator Rights Protocol (UCRP) ownership metadata in various file formats and digital content types.

---

## 📋 **Universal Metadata Fields**

### **Core UCRP Fields**
```
ucrp:content-hash     - SHA-256 hash of normalized content
ucrp:creator          - Creator identifier (platform:username)
ucrp:license          - License type (CC-BY-NC-SA-4.0, etc.)
ucrp:timestamp        - Creation timestamp (ISO 8601)
ucrp:verification     - Verification method/URL
ucrp:record-id        - Unique UCRP record identifier
ucrp:signature        - Cryptographic signature of creator
```

### **Optional Extended Fields**
```
ucrp:platform         - Originating platform name
ucrp:work-id          - Platform-specific work identifier  
ucrp:blockchain       - Blockchain network and transaction
ucrp:rights           - Detailed rights and restrictions
ucrp:derived-from     - Parent work if derivative
ucrp:language         - Content language (ISO 639-1)
```

---

## 🌐 **Web Content (HTML/XHTML)**

### **HTML Meta Tags**
```html
<head>
  <!-- Required UCRP metadata -->
  <meta name="ucrp:content-hash" content="sha256:abc123def456...">
  <meta name="ucrp:creator" content="ao3:username">
  <meta name="ucrp:license" content="CC-BY-NC-SA-4.0">
  <meta name="ucrp:timestamp" content="2025-01-01T00:00:00Z">
  
  <!-- Verification -->
  <meta name="ucrp:verification" content="polygon:0x123abc...">
  <meta name="ucrp:record-id" content="ucrp:12345abcdef">
  
  <!-- Platform information -->
  <meta name="ucrp:platform" content="nuclear-ao3">
  <meta name="ucrp:work-id" content="12345">
  
  <!-- Extended metadata -->
  <meta name="ucrp:language" content="en">
  <meta name="ucrp:fandom" content="Harry Potter">
  <meta name="ucrp:rating" content="Teen And Up Audiences">
  <meta name="ucrp:word-count" content="15000">
  
  <!-- Verification endpoint -->
  <link rel="ucrp:verification" href="https://verify.ucrp.org/12345abcdef">
  <link rel="license" href="https://creativecommons.org/licenses/by-nc-sa/4.0/">
</head>
```

### **JSON-LD Structured Data**
```html
<script type="application/ld+json">
{
  "@context": ["http://schema.org", "https://ucrp.org/context/v1"],
  "@type": "CreativeWork",
  "identifier": "ucrp:12345abcdef",
  "name": "Work Title",
  "author": {
    "@type": "Person",
    "identifier": "ao3:username",
    "name": "Author Display Name"
  },
  "dateCreated": "2025-01-01T00:00:00Z",
  "license": "https://creativecommons.org/licenses/by-nc-sa/4.0/",
  "contentHash": "sha256:abc123def456...",
  "platform": {
    "@type": "WebSite",
    "name": "nuclear-ao3",
    "identifier": "12345"
  },
  "verification": {
    "@type": "DigitalDocument",
    "url": "https://verify.ucrp.org/12345abcdef",
    "blockchain": "polygon:0x123abc..."
  }
}
</script>
```

### **HTML Comments (Backup Method)**
```html
<!-- UCRP-START
{
  "content-hash": "sha256:abc123def456...",
  "creator": "ao3:username", 
  "license": "CC-BY-NC-SA-4.0",
  "timestamp": "2025-01-01T00:00:00Z",
  "verification": "https://verify.ucrp.org/12345abcdef"
}
UCRP-END -->
```

---

## 📸 **Image Files (JPEG/PNG/WebP)**

### **EXIF Metadata (JPEG)**
```
Artist              = "Creator Username"
Copyright           = "© 2025 Creator Name - UCRP:12345abcdef"
ImageDescription    = "ucrp:verification:polygon:0x123abc"
Software            = "UCRP v1.0"
DateTime            = "2025:01:01 00:00:00"
UserComment         = "ucrp:license:CC-BY-NC-SA-4.0"

# Custom EXIF fields (if supported)
UCRP_ContentHash    = "sha256:abc123def456..."
UCRP_Creator        = "platform:username"
UCRP_RecordID       = "ucrp:12345abcdef"
```

### **XMP Metadata (All Formats)**
```xml
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about="">
      <!-- Dublin Core -->
      <dc:creator>Creator Username</dc:creator>
      <dc:rights>© 2025 Creator Name</dc:rights>
      <dc:date>2025-01-01T00:00:00Z</dc:date>
      
      <!-- UCRP Namespace -->
      <ucrp:contentHash>sha256:abc123def456...</ucrp:contentHash>
      <ucrp:creator>platform:username</ucrp:creator>
      <ucrp:license>CC-BY-NC-SA-4.0</ucrp:license>
      <ucrp:recordId>ucrp:12345abcdef</ucrp:recordId>
      <ucrp:verification>polygon:0x123abc...</ucrp:verification>
      <ucrp:platform>nuclear-ao3</ucrp:platform>
      
      <!-- Creative Commons -->
      <cc:license>https://creativecommons.org/licenses/by-nc-sa/4.0/</cc:license>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
```

### **PNG tEXt Chunks**
```
Title           = Work Title
Author          = Creator Username  
Copyright       = © 2025 Creator Name
Creation Time   = 2025-01-01T00:00:00Z
Software        = UCRP v1.0
Comment         = UCRP:12345abcdef
Description     = ucrp:verification:polygon:0x123abc

# Custom PNG tEXt fields
UCRP_Hash       = sha256:abc123def456...
UCRP_Creator    = platform:username
UCRP_License    = CC-BY-NC-SA-4.0
UCRP_Record     = ucrp:12345abcdef
```

### **Steganographic Embedding (Advanced)**
```javascript
// Embed UCRP data in least significant bits
const ucrepData = {
  hash: "sha256:abc123def456...",
  creator: "platform:username",
  timestamp: "2025-01-01T00:00:00Z"
};

// Embed in image LSBs (invisible to human eye)
embedUCRPInImage(imageData, JSON.stringify(ucrepData));
```

---

## 📄 **Text Documents**

### **Plain Text (UTF-8)**
```
---UCRP-METADATA-START---
Content-Hash: sha256:abc123def456...
Creator: platform:username
License: CC-BY-NC-SA-4.0
Timestamp: 2025-01-01T00:00:00Z
Verification: https://verify.ucrp.org/12345abcdef
---UCRP-METADATA-END---

[Actual content follows...]
```

### **Markdown**
```markdown
---
ucrp:
  content-hash: sha256:abc123def456...
  creator: platform:username
  license: CC-BY-NC-SA-4.0
  timestamp: 2025-01-01T00:00:00Z
  verification: https://verify.ucrp.org/12345abcdef
  platform: nuclear-ao3
  work-id: "12345"
---

# Work Title

[Content follows...]
```

### **Word Documents (.docx)**
```xml
<!-- In document.xml Custom XML Parts -->
<ucrp:metadata xmlns:ucrp="https://ucrp.org/schema/v1">
  <ucrp:contentHash>sha256:abc123def456...</ucrp:contentHash>
  <ucrp:creator>platform:username</ucrp:creator>
  <ucrp:license>CC-BY-NC-SA-4.0</ucrp:license>
  <ucrp:timestamp>2025-01-01T00:00:00Z</ucrp:timestamp>
  <ucrp:verification>https://verify.ucrp.org/12345abcdef</ucrp:verification>
</ucrp:metadata>

<!-- In core.xml document properties -->
<dc:creator>Creator Username</dc:creator>
<dc:rights>© 2025 Creator Name</dc:rights>
<dcterms:created>2025-01-01T00:00:00Z</dcterms:created>
```

### **PDF Documents**
```
/Title (Work Title)
/Author (Creator Username)
/Creator (UCRP v1.0)
/CreationDate (D:20250101000000Z)
/Subject (UCRP:12345abcdef)
/Keywords (ucrp:verification:polygon:0x123abc)

# Custom PDF metadata
/UCRP_Hash (sha256:abc123def456...)
/UCRP_Creator (platform:username)
/UCRP_License (CC-BY-NC-SA-4.0)
/UCRP_Record (ucrp:12345abcdef)
```

---

## 🎵 **Audio/Video Files**

### **MP3 ID3v2 Tags**
```
TIT2 = Work Title
TPE1 = Creator Username
TCOP = © 2025 Creator Name
TDRC = 2025-01-01T00:00:00Z
TSSE = UCRP v1.0

# Custom frames
TXXX[UCRP_Hash] = sha256:abc123def456...
TXXX[UCRP_Creator] = platform:username
TXXX[UCRP_License] = CC-BY-NC-SA-4.0
TXXX[UCRP_Record] = ucrp:12345abcdef
WXXX[UCRP_Verify] = https://verify.ucrp.org/12345abcdef
```

### **MP4/M4A Metadata**
```
©nam = Work Title
©ART = Creator Username
©day = 2025-01-01T00:00:00Z
©too = UCRP v1.0

# Custom atoms
UCRP = {
  "hash": "sha256:abc123def456...",
  "creator": "platform:username", 
  "license": "CC-BY-NC-SA-4.0",
  "record": "ucrp:12345abcdef"
}
```

### **WebM/MKV**
```xml
<Tags>
  <Tag>
    <Simple>
      <Name>TITLE</Name>
      <String>Work Title</String>
    </Simple>
    <Simple>
      <Name>ARTIST</Name>
      <String>Creator Username</String>
    </Simple>
    <Simple>
      <Name>DATE_RECORDED</Name>
      <String>2025-01-01T00:00:00Z</String>
    </Simple>
    <Simple>
      <Name>UCRP_HASH</Name>
      <String>sha256:abc123def456...</String>
    </Simple>
    <Simple>
      <Name>UCRP_CREATOR</Name>
      <String>platform:username</String>
    </Simple>
    <Simple>
      <Name>UCRP_LICENSE</Name>
      <String>CC-BY-NC-SA-4.0</String>
    </Simple>
    <Simple>
      <Name>UCRP_VERIFICATION</Name>
      <String>https://verify.ucrp.org/12345abcdef</String>
    </Simple>
  </Tag>
</Tags>
```

---

## 📊 **Data Formats**

### **JSON**
```json
{
  "_ucrp": {
    "content-hash": "sha256:abc123def456...",
    "creator": "platform:username",
    "license": "CC-BY-NC-SA-4.0", 
    "timestamp": "2025-01-01T00:00:00Z",
    "verification": "https://verify.ucrp.org/12345abcdef",
    "record-id": "ucrp:12345abcdef"
  },
  "data": {
    "title": "Work Title",
    "content": "Actual work content..."
  }
}
```

### **XML**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<document xmlns:ucrp="https://ucrp.org/schema/v1">
  <ucrp:metadata>
    <ucrp:contentHash>sha256:abc123def456...</ucrp:contentHash>
    <ucrp:creator>platform:username</ucrp:creator>
    <ucrp:license>CC-BY-NC-SA-4.0</ucrp:license>
    <ucrp:timestamp>2025-01-01T00:00:00Z</ucrp:timestamp>
    <ucrp:verification>https://verify.ucrp.org/12345abcdef</ucrp:verification>
  </ucrp:metadata>
  <content>
    <title>Work Title</title>
    <body>Actual work content...</body>
  </content>
</document>
```

### **CSV (with metadata header)**
```csv
# UCRP-METADATA: {"hash":"sha256:abc123...","creator":"platform:username","license":"CC-BY-NC-SA-4.0"}
# Verification: https://verify.ucrp.org/12345abcdef
Title,Author,Content
"Work Title","Creator Username","Actual content..."
```

---

## 🛡️ **Security Considerations**

### **Content Hash Calculation**
```javascript
// Normalize content before hashing
function calculateContentHash(content, type) {
  let normalized;
  
  switch(type) {
    case 'text':
      // Normalize line endings and encoding
      normalized = content
        .replace(/\r\n/g, '\n')
        .replace(/\r/g, '\n')
        .trim();
      break;
      
    case 'image':
      // Hash pixel data, exclude metadata
      normalized = extractPixelData(content);
      break;
      
    case 'html':
      // Normalize whitespace, exclude UCRP metadata
      normalized = normalizeHTML(content);
      break;
  }
  
  return sha256(normalized);
}
```

### **Metadata Integrity**
```javascript
// Sign metadata with creator's private key
function signMetadata(metadata, privateKey) {
  const canonicalized = JSON.stringify(metadata, Object.keys(metadata).sort());
  return sign(canonicalized, privateKey);
}

// Verify signature
function verifyMetadata(metadata, signature, publicKey) {
  const canonicalized = JSON.stringify(metadata, Object.keys(metadata).sort());
  return verify(canonicalized, signature, publicKey);
}
```

### **Anti-Tampering Measures**
1. **Multiple Embedding Locations** - Store metadata in multiple places
2. **Redundant Verification** - Cross-reference with blockchain records
3. **Checksum Validation** - Verify metadata integrity
4. **Steganographic Backup** - Hidden embedding for critical data

---

## 🔧 **Implementation Libraries**

### **JavaScript/Node.js**
```javascript
import { UCRPMetadata } from 'ucrp-metadata';

// Embed metadata in HTML
const html = UCRPMetadata.embedHTML(content, {
  creator: 'ao3:username',
  license: 'CC-BY-NC-SA-4.0',
  platform: { name: 'nuclear-ao3', workId: '12345' }
});

// Extract metadata
const metadata = UCRPMetadata.extractHTML(html);
console.log(metadata.creator); // 'ao3:username'
```

### **Python**
```python
from ucrp_metadata import UCRPMetadata

# Embed in image EXIF
metadata = UCRPMetadata(
    creator='ao3:username',
    license='CC-BY-NC-SA-4.0',
    platform={'name': 'nuclear-ao3', 'work_id': '12345'}
)
metadata.embed_image('input.jpg', 'output.jpg')

# Extract from image
extracted = UCRPMetadata.extract_image('output.jpg')
print(extracted.creator)  # 'ao3:username'
```

### **Go**
```go
package main

import "github.com/ucrp/metadata-go"

func main() {
    meta := &ucrp.Metadata{
        Creator:  "ao3:username",
        License:  "CC-BY-NC-SA-4.0",
        Platform: ucrp.Platform{Name: "nuclear-ao3", WorkID: "12345"},
    }
    
    // Embed in PDF
    err := meta.EmbedPDF("input.pdf", "output.pdf")
    if err != nil {
        panic(err)
    }
    
    // Extract from PDF
    extracted, err := ucrp.ExtractPDF("output.pdf")
    if err != nil {
        panic(err)
    }
    fmt.Println(extracted.Creator) // ao3:username
}
```

---

## 📋 **Best Practices**

### **For Platform Developers**
1. **Always embed basic metadata** (creator, license, timestamp)
2. **Use multiple embedding methods** for redundancy
3. **Calculate content hash correctly** using normalization
4. **Preserve metadata** during format conversions
5. **Validate metadata** before embedding

### **For Content Creators**
1. **Choose appropriate licensing** for your content
2. **Verify metadata embedding** in exported files
3. **Keep verification URLs** accessible
4. **Update contact information** if creator identity changes
5. **Understand your rights** under chosen licenses

### **Cross-Platform Compatibility**
- Use standard metadata fields when possible
- Provide fallback methods for unsupported formats
- Test metadata preservation across platforms
- Document any custom extensions clearly

This comprehensive metadata embedding standard ensures Universal Creator Rights Protocol information can be preserved and verified across all major digital content formats while maintaining compatibility with existing metadata standards.