const axios = require('axios');

class KoskidexClient {
    constructor(host = 'http://localhost:7700', apiKey = '') {
        this.client = axios.create({
            baseURL: host,
            headers: apiKey ? { 'Authorization': `Bearer ${apiKey}` } : {}
        });
    }

    async createIndex(name) {
        const res = await this.client.post('/indexes', { name });
        return res.data;
    }

    async addDocuments(indexName, documents) {
        const res = await this.client.post(`/indexes/${indexName}/documents`, documents);
        return res.data;
    }

    async search(indexName, query) {
        const res = await this.client.get(`/indexes/${indexName}/search`, {
            params: { q: query }
        });
        return res.data;
    }
}

// ============== USAGE EXAMPLE ==============
async function runExample() {
    const koski = new KoskidexClient('http://localhost:7700');

    try {
        console.log("Creating 'products' index...");
        await koski.createIndex('products');
    } catch (e) {
        console.log("Index already exists or error:", e.response?.data || e.message);
    }

    console.log("\nAdding documents in batch...");
    await koski.addDocuments('products', [
        { id: '100', name: 'MacBook Pro 14', price: 1999, category: 'Laptops' },
        { id: '101', name: 'iPhone 15 Pro', price: 999, category: 'Smartphones' },
        { id: '102', name: 'iPad Air', price: 599, category: 'Tablets' }
    ]);

    console.log("\nSearching: 'macbok' (Typo test)...");
    const results = await koski.search('products', 'macbok');
    
    console.log("\nResults found:");
    console.log(JSON.stringify(results, null, 2));
}

// Uncomment to test:
// runExample();

module.exports = KoskidexClient;
