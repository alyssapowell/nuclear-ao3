# Universal Creator Rights Protocol (UCRP) v1.0
## Technical Specification

### 🎯 **Protocol Overview**

The Universal Creator Rights Protocol (UCRP) defines a standardized, blockchain-agnostic method for establishing, tracking, and verifying digital asset ownership across platforms and mediums.

---

## 📋 **Core Data Structures**

### **1. Ownership Record**
```json
{
  "@context": "https://ucrp.org/v1",
  "@type": "CreativeWork",
  "ucrp_version": "1.0",
  "record_id": "ucrp:12345abcdef",
  "content": {
    "hash": "sha256:abc123...",
    "algorithm": "sha256",
    "size": 150000,
    "mimeType": "text/html",
    "language": "en-US"
  },
  "creator": {
    "identifier": "platform:username",
    "publicKey": "0x123abc...",
    "verification": {
      "method": "blockchain|email|platform",
      "signature": "0x456def...",
      "timestamp": "2025-01-01T00:00:00Z"
    }
  },
  "timestamp": {
    "created": "2025-01-01T00:00:00Z",
    "registered": "2025-01-01T00:05:00Z",
    "lastModified": "2025-01-01T00:05:00Z"
  },
  "rights": {
    "license": "CC-BY-NC-SA-4.0",
    "permissions": {
      "attribution": "required",
      "commercialUse": "prohibited",
      "derivatives": "allowed",
      "sharing": "allowed",
      "transformativeWorks": "encouraged"
    },
    "restrictions": {
      "aiTraining": "prohibited",
      "nftMinting": "prohibited",
      "commercialAdaptation": "contact_creator"
    }
  },
  "platform": {
    "name": "nuclear-ao3",
    "workId": "12345",
    "url": "https://nuclear-ao3.org/works/12345",
    "metadata": {
      "fandom": "Harry Potter",
      "rating": "Teen And Up Audiences",
      "wordCount": 15000
    }
  },
  "blockchain": {
    "network": "polygon",
    "transactionHash": "0x456def...",
    "blockNumber": 12345678,
    "contractAddress": "0x789abc...",
    "tokenId": "12345"
  },
  "verification": {
    "nodes": [
      "https://verify1.ucrp.org",
      "https://verify2.ucrp.org"
    ],
    "consensus": 0.95,
    "lastVerified": "2025-01-01T12:00:00Z"
  }
}
```

### **2. Transfer Record**
```json
{
  "@context": "https://ucrp.org/v1",
  "@type": "OwnershipTransfer",
  "recordId": "ucrp:12345abcdef",
  "from": "platform:creator1",
  "to": "platform:creator2",
  "transferType": "sale|gift|license|inheritance",
  "timestamp": "2025-01-01T00:00:00Z",
  "blockchain": {
    "transactionHash": "0x789def...",
    "network": "polygon"
  },
  "terms": {
    "price": 100.00,
    "currency": "USD",
    "retainedRights": ["attribution", "revert"],
    "expirationDate": "2030-01-01T00:00:00Z"
  }
}
```

---

## 🔐 **Cryptographic Standards**

### **Content Hashing**
- **Algorithm**: SHA-256 (required), SHA-3, Blake2b (optional)
- **Normalization**: UTF-8 encoding, CRLF → LF conversion
- **Metadata Exclusion**: Hash only core content, exclude metadata

### **Digital Signatures** 
- **ECDSA**: secp256k1 (Ethereum compatible)
- **EdDSA**: ed25519 (recommended for new implementations)
- **RSA**: 2048-bit minimum (legacy support)

### **Key Derivation**
```
Private Key → Public Key → Address/Identifier → UCRP ID
```

---

## 🌐 **Network Protocol**

### **Verification Network Architecture**

#### **Node Types**
1. **Registry Nodes** - Store and serve ownership records
2. **Verification Nodes** - Validate ownership claims
3. **Arbitration Nodes** - Resolve disputes
4. **Gateway Nodes** - Provide public API access

#### **Consensus Mechanism**
- **Threshold**: 67% agreement for validity
- **Voting Weight**: Equal weight per node
- **Dispute Resolution**: Human arbitrators for complex cases

### **API Endpoints**

#### **1. Register New Work**
```
POST /v1/register
Content-Type: application/json

{
  "content": "base64_encoded_content_or_hash",
  "creator": "verified_creator_id",
  "license": "CC-BY-NC-SA-4.0",
  "platform": {
    "name": "nuclear-ao3",
    "workId": "12345"
  },
  "signature": "creator_signature"
}

Response:
{
  "recordId": "ucrp:12345abcdef",
  "status": "pending|verified|rejected",
  "verificationUrl": "https://verify.ucrp.org/12345abcdef",
  "expiresAt": "2025-01-01T01:00:00Z"
}
```

#### **2. Verify Ownership**
```
GET /v1/verify/{content_hash_or_record_id}

Response:
{
  "verified": true,
  "confidence": 0.98,
  "recordId": "ucrp:12345abcdef",
  "creator": "platform:username",
  "timestamp": "2025-01-01T00:00:00Z",
  "license": "CC-BY-NC-SA-4.0",
  "platform": {
    "name": "nuclear-ao3",
    "workId": "12345",
    "url": "https://nuclear-ao3.org/works/12345"
  },
  "blockchain": {
    "network": "polygon",
    "transactionHash": "0x456def..."
  },
  "verificationNodes": [
    {
      "url": "https://verify1.ucrp.org",
      "status": "verified",
      "timestamp": "2025-01-01T00:05:00Z"
    }
  ]
}
```

#### **3. Query by Creator**
```
GET /v1/creator/{creator_id}?limit=50&offset=0

Response:
{
  "total": 150,
  "works": [
    {
      "recordId": "ucrp:12345abcdef",
      "contentHash": "sha256:abc123...",
      "title": "Work Title",
      "platform": "nuclear-ao3",
      "timestamp": "2025-01-01T00:00:00Z"
    }
  ]
}
```

#### **4. Report Dispute**
```
POST /v1/dispute
{
  "recordId": "ucrp:12345abcdef",
  "claimant": "disputing_creator_id",
  "disputeType": "false_claim|unauthorized_use|impersonation",
  "evidence": [
    {
      "type": "blockchain_transaction",
      "data": "0x123abc..."
    },
    {
      "type": "platform_proof",
      "url": "https://platform.com/proof"
    }
  ],
  "description": "Detailed explanation of dispute"
}
```

---

## 📊 **Implementation Levels**

### **Level 1: Basic Metadata (Entry Level)**
- **Requirements**: Embed UCRP metadata in content
- **Technology**: HTML meta tags, JSON-LD, EXIF
- **Cost**: Free
- **Protection**: Attribution tracking, basic verification

### **Level 2: Federated Verification (Standard)**
- **Requirements**: Register with UCRP network
- **Technology**: API integration, periodic verification
- **Cost**: $0.01-0.10 per work
- **Protection**: Cross-platform verification, dispute resolution

### **Level 3: Blockchain Registration (Premium)**
- **Requirements**: Blockchain transaction for permanence
- **Technology**: Smart contracts, IPFS storage
- **Cost**: $0.50-5.00 per work (depending on chain)
- **Protection**: Immutable proof, legal evidence, transferable ownership

### **Level 4: Full Rights Management (Enterprise)**
- **Requirements**: Complete licensing and transfer tracking
- **Technology**: Advanced smart contracts, legal integration
- **Cost**: $5-50 per work + percentage of transfers
- **Protection**: Commercial licensing, royalty tracking, legal enforcement

---

## 🔄 **Lifecycle Management**

### **Work Registration Flow**
```
1. Creator submits work → 2. Content hash generated → 
3. Signature verification → 4. Network consensus → 
5. Record creation → 6. Blockchain registration (optional) →
7. Verification complete
```

### **Update Procedures**
- **Content Updates**: New hash creates derivative record
- **Rights Changes**: Signed update by creator
- **Transfer**: Multi-signature transaction with both parties
- **Dispute Resolution**: Arbitration panel decision

### **Data Retention**
- **Active Records**: Maintained indefinitely
- **Disputed Records**: Archived after resolution
- **Abandoned Records**: Grace period then archival
- **Legal Requests**: Compliance with jurisdiction requirements

---

## 🛡️ **Security Considerations**

### **Attack Vectors & Mitigations**

#### **1. False Ownership Claims**
- **Mitigation**: Multi-source verification, reputation systems
- **Detection**: Cross-platform timestamp analysis

#### **2. Key Compromise**
- **Mitigation**: Key rotation, multi-signature requirements
- **Recovery**: Social recovery through verified contacts

#### **3. Platform Manipulation**
- **Mitigation**: Multiple verification sources, blockchain backup
- **Detection**: Consensus disagreement alerts

#### **4. Network Attacks**
- **Mitigation**: Distributed nodes, Byzantine fault tolerance
- **Monitoring**: Real-time consensus monitoring

### **Privacy Protection**
- **Pseudonymous Creation**: Support for anonymous creators
- **Selective Disclosure**: Private metadata with public proof
- **Right to Erasure**: Archival without deletion for legal compliance

---

## 📏 **Compliance & Standards**

### **Legal Frameworks**
- **Copyright Law**: Compatible with DMCA, EU Copyright Directive
- **Privacy Law**: GDPR, CCPA compliant data handling
- **Consumer Protection**: Clear terms, dispute resolution

### **Technical Standards**
- **ISO 8601**: Timestamp formatting
- **RFC 3339**: Date/time representation
- **JSON-LD**: Structured data format
- **W3C DID**: Decentralized identifier compatibility

### **Industry Integration**
- **Creative Commons**: Direct license compatibility
- **ORCID**: Academic identifier support  
- **DOI**: Research publication integration
- **ISBN/ISSN**: Publishing industry standards

---

## 🔧 **Reference Implementation**

### **Node.js Client Library**
```javascript
import { UCRPClient } from 'ucrp-js';

const client = new UCRPClient({
  network: 'mainnet',
  nodes: ['https://node1.ucrp.org', 'https://node2.ucrp.org']
});

// Register new work
const record = await client.register({
  content: workContent,
  creator: authenticatedCreator,
  license: 'CC-BY-NC-SA-4.0',
  platform: { name: 'nuclear-ao3', workId: '12345' }
});

// Verify ownership
const verification = await client.verify(contentHash);
console.log(`Verified: ${verification.verified}, Creator: ${verification.creator}`);
```

### **Python Implementation**
```python
from ucrp import UCRPClient

client = UCRPClient(
    network='mainnet',
    nodes=['https://node1.ucrp.org', 'https://node2.ucrp.org']
)

# Register work
record = client.register(
    content=work_content,
    creator=authenticated_creator,
    license='CC-BY-NC-SA-4.0',
    platform={'name': 'nuclear-ao3', 'work_id': '12345'}
)

# Verify ownership
verification = client.verify(content_hash)
print(f"Verified: {verification['verified']}, Creator: {verification['creator']}")
```

---

This protocol specification enables universal digital rights management while maintaining creator control and platform independence. Implementation can be gradual, starting with basic metadata and evolving to full blockchain integration based on platform needs and creator preferences.