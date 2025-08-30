# adobe_pdf - Advanced Integration

Complex scenarios with error handling and optimization for Adobe PDF Services for document processing

**Complexity Level:** advanced
**Generated:** 2025-08-30T00:29:20.885Z

## Examples

### Batch Operations

Performing batch operations efficiently

**Category:** batch

```javascript
// Batch operations with rate limiting
class Adobe_pdfBatchProcessor {
  constructor(mcpClient, options = {}) {
    this.client = mcpClient;
    this.batchSize = options.batchSize || 10;
    this.delayMs = options.delayMs || 100;
    this.maxConcurrent = options.maxConcurrent || 3;
  }

  async processBatch(items) {
    const results = [];
    const batches = this.createBatches(items, this.batchSize);
    
    for (const batch of batches) {
      const batchPromises = batch.map(item => 
        this.processItem(item).catch(error => ({ error, item }))
      );
      
      const batchResults = await Promise.all(batchPromises);
      results.push(...batchResults);
      
      // Rate limiting delay
      if (batches.indexOf(batch) < batches.length - 1) {
        await this.delay(this.delayMs);
      }
    }
    
    return results;
  }

  async processItem(item) {
    const request = {
      jsonrpc: "2.0",
      id: `batch-${Date.now()}-${Math.random()}`,
      method: "tools/call",
      params: {
        name: "adobe_pdf.createAsset",
        arguments: {
          ...{
          "body": {
                    "mediaType": "example-value"
          }
},
          ...item
        }
      }
    };
    
    return await this.client.request(request);
  }

  createBatches(items, size) {
    const batches = [];
    for (let i = 0; i < items.length; i += size) {
      batches.push(items.slice(i, i + size));
    }
    return batches;
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Usage example
const processor = new Adobe_pdfBatchProcessor(mcpClient, {
  batchSize: 5,
  delayMs: 200,
  maxConcurrent: 2
});

const items = [
  { id: 1, name: "Item 1" },
  { id: 2, name: "Item 2" },
  // ... more items
];

const results = await processor.processBatch(items);
console.log(`Processed ${results.length} items`);
```

### Rate Limiting & Retry

Handling rate limits with retry logic

**Category:** resilience

```javascript
// Retry logic with exponential backoff
class RetryableClient {
  constructor(mcpClient, options = {}) {
    this.client = mcpClient;
    this.maxRetries = options.maxRetries || 3;
    this.baseDelay = options.baseDelay || 1000;
    this.maxDelay = options.maxDelay || 10000;
  }

  async callWithRetry(toolName, arguments, options = {}) {
    const maxRetries = options.maxRetries || this.maxRetries;
    let lastError;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        const request = {
          jsonrpc: "2.0",
          id: `retry-${Date.now()}-${attempt}`,
          method: "tools/call",
          params: { name: toolName, arguments }
        };

        const response = await this.client.request(request);
        
        // Success - return result
        return response;
        
      } catch (error) {
        lastError = error;
        
        // Don't retry on certain errors
        if (this.isNonRetryableError(error)) {
          throw error;
        }
        
        // Don't retry on last attempt
        if (attempt === maxRetries) {
          break;
        }
        
        // Calculate delay with exponential backoff
        const delay = Math.min(
          this.baseDelay * Math.pow(2, attempt),
          this.maxDelay
        );
        
        console.log(`Attempt ${attempt + 1} failed, retrying in ${delay}ms...`);
        await this.delay(delay);
      }
    }
    
    throw new Error(`Failed after ${maxRetries + 1} attempts: ${lastError.message}`);
  }

  isNonRetryableError(error) {
    // Don't retry on authentication or validation errors
    const nonRetryableCodes = [400, 401, 403, 404, 422];
    return nonRetryableCodes.includes(error.statusCode);
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Usage example
const retryClient = new RetryableClient(mcpClient, {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 8000
});

try {
  const result = await retryClient.callWithRetry(
    "adobe_pdf.createAsset",
    {
    "body": {
        "mediaType": "example-value"
    }
}
  );
  console.log("Operation succeeded:", result);
} catch (error) {
  console.error("Operation failed permanently:", error.message);
}
```

### Performance Optimization

Optimizing API calls for better performance

**Category:** optimization

```javascript
// Performance optimization techniques
class OptimizedClient {
  constructor(mcpClient) {
    this.client = mcpClient;
    this.cache = new Map();
    this.requestQueue = [];
    this.processing = false;
  }

  // Request batching
  async batchRequests(requests, batchSize = 10) {
    const batches = [];
    for (let i = 0; i < requests.length; i += batchSize) {
      batches.push(requests.slice(i, i + batchSize));
    }

    const results = [];
    for (const batch of batches) {
      const batchResults = await Promise.all(
        batch.map(req => this.client.request(req))
      );
      results.push(...batchResults);
    }

    return results;
  }

  // Response caching
  async cachedCall(toolName, arguments, ttlMs = 300000) {
    const cacheKey = `${toolName}:${JSON.stringify(arguments)}`;
    const cached = this.cache.get(cacheKey);
    
    if (cached && Date.now() - cached.timestamp < ttlMs) {
      return cached.data;
    }

    const result = await this.client.request({
      jsonrpc: "2.0",
      id: `cached-${Date.now()}`,
      method: "tools/call",
      params: { name: toolName, arguments }
    });

    this.cache.set(cacheKey, {
      data: result,
      timestamp: Date.now()
    });

    return result;
  }

  // Request deduplication
  async deduplicatedCall(toolName, arguments) {
    const key = `${toolName}:${JSON.stringify(arguments)}`;
    
    if (this.pendingRequests?.has(key)) {
      return await this.pendingRequests.get(key);
    }

    if (!this.pendingRequests) {
      this.pendingRequests = new Map();
    }

    const promise = this.client.request({
      jsonrpc: "2.0",
      id: `dedup-${Date.now()}`,
      method: "tools/call",
      params: { name: toolName, arguments }
    }).finally(() => {
      this.pendingRequests.delete(key);
    });

    this.pendingRequests.set(key, promise);
    return await promise;
  }
}

// Usage example
const optimizedClient = new OptimizedClient(mcpClient);

// Batch multiple requests
const requests = [
  { jsonrpc: "2.0", id: "1", method: "tools/call", params: { name: "adobe_pdf.createAsset", arguments: {} }},
  { jsonrpc: "2.0", id: "2", method: "tools/call", params: { name: "adobe_pdf.combinePDF", arguments: {} }}
];

const batchResults = await optimizedClient.batchRequests(requests);

// Use caching for repeated calls
const cachedResult = await optimizedClient.cachedCall(
  "adobe_pdf.createAsset",
  {},
  600000 // 10 minute cache
);
```

