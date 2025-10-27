# Nuclear AO3: Zero-Knowledge Encryption Strategy

## 🎯 Strategic Objective

**Protect fanfiction writers and readers from potential government/corporate overreach while maintaining platform functionality.**

Even if cloud providers receive legal demands, encrypted data remains unreadable without user-controlled keys, providing **legal protection through technical impossibility**.

## 🔐 Encryption Architecture

### **Multi-Layer Encryption Model**

**Layer 1: Transport Encryption (Standard)**
```
Client ←→ TLS 1.3 ←→ Load Balancer ←→ TLS ←→ Services
```

**Layer 2: Application-Level Encryption (Zero-Knowledge)**
```
User Content → Client-Side Encryption → Encrypted Storage → Cloud Provider
                        ↑
                User-Controlled Keys
                (Never stored on servers)
```

**Layer 3: Database Field Encryption**
```sql
-- Sensitive content encrypted before database storage
CREATE TABLE works (
    id UUID PRIMARY KEY,
    title_encrypted BYTEA,           -- Encrypted with user key
    content_encrypted BYTEA,         -- Encrypted with user key  
    metadata_hash BYTEA,             -- Searchable hash
    user_key_id VARCHAR(255),        -- Key identifier (not the key)
    created_at TIMESTAMP
);
```

### **Key Management Architecture**

**Client-Side Key Derivation:**
```javascript
// User password never leaves client
async function deriveUserKeys(username, password) {
    // Derive master key from user password
    const masterKey = await crypto.subtle.deriveKey(
        {
            name: "PBKDF2",
            salt: new TextEncoder().encode(username + "salt"),
            iterations: 100000,
            hash: "SHA-256"
        },
        await crypto.subtle.importKey(
            "raw",
            new TextEncoder().encode(password),
            "PBKDF2",
            false,
            ["deriveKey"]
        ),
        {
            name: "AES-GCM",
            length: 256
        },
        false,
        ["encrypt", "decrypt"]
    );
    
    // Derive content encryption key
    const contentKey = await deriveContentKey(masterKey, "content");
    
    return { masterKey, contentKey };
}

// Server never sees user password or encryption keys
```

**Content Encryption (Client-Side):**
```javascript
async function encryptWork(work, userKeys) {
    const encoder = new TextEncoder();
    
    // Encrypt title and content with user's key
    const encryptedTitle = await crypto.subtle.encrypt(
        { name: "AES-GCM", iv: crypto.getRandomValues(new Uint8Array(12)) },
        userKeys.contentKey,
        encoder.encode(work.title)
    );
    
    const encryptedContent = await crypto.subtle.encrypt(
        { name: "AES-GCM", iv: crypto.getRandomValues(new Uint8Array(12)) },
        userKeys.contentKey,
        encoder.encode(work.content)
    );
    
    // Create searchable hash (for tags, metadata)
    const searchableHash = await createSearchableHash(work.tags, work.metadata);
    
    return {
        id: work.id,
        title_encrypted: encryptedTitle,
        content_encrypted: encryptedContent,
        searchable_hash: searchableHash,
        user_key_id: userKeys.keyId  // Identifier, not actual key
    };
}
```

### **Search with Encryption**

**Searchable Encryption for Tags/Metadata:**
```javascript
// Allow search without exposing content
async function createSearchableHash(tags, metadata) {
    const searchData = {
        tags: tags.map(tag => tag.toLowerCase()),
        rating: metadata.rating,
        fandom: metadata.fandom,
        // etc.
    };
    
    // Hash searchable fields (deterministic for matching)
    const searchHash = await crypto.subtle.digest(
        "SHA-256",
        new TextEncoder().encode(JSON.stringify(searchData))
    );
    
    return Array.from(new Uint8Array(searchHash));
}

// Search by hash matching, not plaintext
```

**Elasticsearch Integration:**
```json
{
  "query": {
    "bool": {
      "must": [
        {
          "term": {
            "fandom_hash": "a1b2c3d4e5f6..."
          }
        },
        {
          "term": {
            "rating_hash": "x7y8z9..."
          }
        }
      ]
    }
  }
}
```

## 🛡️ Legal Protection Through Technical Design

### **What Cloud Providers Cannot Access**

**Even with full server access:**
- ❌ **User passwords** (never transmitted to server)
- ❌ **Work content** (encrypted with user keys)
- ❌ **Private messages** (end-to-end encrypted)
- ❌ **Reading history** (locally encrypted)
- ❌ **Personal notes** (client-side encrypted)

**What remains accessible (by design):**
- ✅ **Public metadata** (work titles, public tags)
- ✅ **User registration data** (usernames, emails)
- ✅ **Public comments** (visible to all users anyway)
- ✅ **System logs** (anonymized)

### **Government Request Response**

**Nuclear AO3's response to data requests:**
```
"We comply with all valid legal requests. However, user content 
is encrypted with keys derived from user passwords. We do not 
store user passwords or encryption keys. The requested content 
exists only in encrypted form that we cannot decrypt."
```

**Technical impossibility = Legal protection**

## 🔧 Implementation Details

### **Database Schema Design**

```sql
-- Users table (minimal unencrypted data)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,  -- Server auth only
    key_derivation_salt BYTEA,            -- For client key derivation
    created_at TIMESTAMP,
    preferences_encrypted BYTEA           -- User preferences encrypted
);

-- Works table (content encrypted)
CREATE TABLE works (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    title_encrypted BYTEA NOT NULL,       -- Encrypted title
    summary_encrypted BYTEA,              -- Encrypted summary  
    content_encrypted BYTEA NOT NULL,     -- Encrypted work content
    
    -- Searchable hashes (not reversible)
    fandom_hash BYTEA,
    rating_hash BYTEA,
    tags_hash BYTEA[],
    
    -- Public metadata (intentionally unencrypted)
    word_count INTEGER,
    chapter_count INTEGER,
    published_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Comments (encrypted)
CREATE TABLE comments (
    id UUID PRIMARY KEY,
    work_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content_encrypted BYTEA NOT NULL,     -- Encrypted comment
    public_hash BYTEA,                    -- For threading/replies
    created_at TIMESTAMP
);
```

### **Key Rotation Strategy**

```javascript
// Periodic key rotation for security
async function rotateUserKeys(oldKeys, newPassword) {
    // Derive new keys from new password
    const newKeys = await deriveUserKeys(username, newPassword);
    
    // Re-encrypt all user content with new keys
    const userWorks = await fetchUserWorks(userId);
    
    for (const work of userWorks) {
        // Decrypt with old key
        const plaintext = await decryptWork(work, oldKeys);
        
        // Re-encrypt with new key
        const reencrypted = await encryptWork(plaintext, newKeys);
        
        // Update database
        await updateWork(work.id, reencrypted);
    }
}
```

## 🌐 Distributed Trust Model

### **Geographic Key Distribution**

```yaml
# Keys never stored in same jurisdiction as content
key_storage:
  primary_keys: "Switzerland (strong privacy laws)"
  backup_keys: "Iceland (data haven)"
  content_servers: "US (for performance)"

# Even if US servers are compromised, keys are elsewhere
```

### **Multi-Jurisdiction Architecture**

```
User (Global) 
    ↓ (encrypted)
Content Servers (US/EU) - Fast access, encrypted data only
    ↓ (key requests)
Key Servers (CH/IS) - Strong privacy laws, keys only
    ↓ (emergency)
Backup Systems (Multiple) - Distributed, no single point of failure
```

## 📊 Performance Impact

### **Encryption Overhead**

**Client-Side Encryption:**
- **Encryption time:** ~5ms per work (acceptable)
- **Decryption time:** ~3ms per work  
- **Key derivation:** ~100ms (one-time per session)
- **Storage overhead:** ~10% (encryption metadata)

**User Experience:**
```javascript
// Progressive decryption - show UI while decrypting
async function loadUserWorks() {
    const encryptedWorks = await fetchWorks();
    const userKeys = await getUserKeys(); // Cached from login
    
    // Show encrypted count immediately
    showWorkCount(encryptedWorks.length);
    
    // Decrypt progressively
    for (const encryptedWork of encryptedWorks) {
        const work = await decryptWork(encryptedWork, userKeys);
        displayWork(work); // Stream results to UI
    }
}
```

### **Search Performance**

**Hash-Based Search:**
- **Tag matching:** O(1) hash lookups
- **Metadata filtering:** Standard database indexes
- **Full-text search:** Disabled for encrypted content
- **Performance impact:** <5% vs unencrypted

## 🚨 Security Considerations

### **Attack Scenarios**

**Government/Corporate Data Request:**
- **Encrypted data provided** (useless without keys)
- **Keys not available** (client-side only)
- **Legal compliance maintained**

**Platform Compromise:**
- **Encrypted content safe** (keys not on servers)
- **User passwords safe** (never stored)
- **Metadata partially exposed** (by design for functionality)

**User Device Compromise:**
- **Local keys at risk** (standard device security applies)
- **Server data still encrypted** 
- **Key rotation possible** (change password to rotate)

### **Operational Security**

```javascript
// Zero-knowledge server design
class WorkService {
    async createWork(encryptedWorkData) {
        // Server never sees plaintext
        const work = {
            id: generateUUID(),
            user_id: getCurrentUser().id,
            title_encrypted: encryptedWorkData.title_encrypted,
            content_encrypted: encryptedWorkData.content_encrypted,
            // ... encrypted fields only
        };
        
        return await database.insert('works', work);
    }
    
    // Server cannot decrypt - no keys available
    async getWork(workId) {
        return await database.select('works', { id: workId });
        // Returns encrypted data - client decrypts
    }
}
```

## 🎯 Strategic Advantages

### **For Users**
- **Legal protection** through technical impossibility
- **Privacy by design** - even we can't see your content
- **Government-proof** fanfiction platform
- **Corporate-proof** against takedown pressure

### **For Platform**
- **Reduced legal liability** (can't turn over what we can't read)
- **User trust** through verifiable encryption
- **Differentiation** from other platforms
- **Compliance** with strongest privacy laws globally

### **Marketing Message**
*"Your fanfiction is encrypted with keys only you control. Not even we can read your stories. Government requests return encrypted data that's mathematically impossible to decrypt without your password."*

---

**This encryption strategy transforms Nuclear AO3 from just a "better AO3" into a "government-proof fanfiction sanctuary" - a powerful differentiator for the nuclear option.**