/* eslint-env node */
const axios = require("axios");

/**
 * KoskidexClient - Node.js client for the Koskidex search engine.
 */
class KoskidexClient {
  /**
   * @param {string} host - The base URL of the Koskidex server.
   * @param {string} apiKey - Optional API key for authentication.
   */
  constructor(host = "http://localhost:7700", apiKey = "") {
    this.client = axios.create({
      baseURL: host,
      headers: apiKey ? { Authorization: `Bearer ${apiKey}` } : {},
    });
  }

  /**
   * @param {string} name - The name of the index to create.
   * @returns {Promise<any>}
   */
  async createIndex(name) {
    const res = await this.client.post("/indexes", { name });
    return res.data;
  }

  /**
   * @param {string} indexName - The name of the index.
   * @param {Array<Object>} documents - Array of documents to add.
   * @returns {Promise<any>}
   */
  async addDocuments(indexName, documents) {
    const res = await this.client.post(
      `/indexes/${indexName}/documents`,
      documents,
    );
    return res.data;
  }

  /**
   * @param {string} indexName - The name of the index.
   * @param {Object} document - A single document object to add.
   * @returns {Promise<any>}
   */
  async addDocument(indexName, document) {
    const res = await this.client.post(
      `/indexes/${indexName}/documents`,
      document,
    );
    return res.data;
  }

  /**
   * @param {string} indexName - The name of the index.
   * @param {string} query - The search query string.
   * @returns {Promise<any>}
   */
  async search(indexName, query) {
    const res = await this.client.get(`/indexes/${indexName}/search`, {
      params: { q: query },
    });
    return res.data;
  }
}

// ============== USAGE EXAMPLE ==============
async function runExample() {
  const koski = new KoskidexClient("http://localhost:7700");

  try {
    console.log("Creating 'products' index...");
    await koski.createIndex("products");
  } catch (/** @type {any} */ e) {
    console.log(
      "Index already exists or error:",
      e.response?.data || e.message,
    );
  }

  console.log("\nAdding documents in batch...");
  await koski.addDocuments("products", [
    { id: "100", name: "MacBook Pro 14", price: 1999, category: "Laptops" },
    { id: "101", name: "iPhone 15 Pro", price: 999, category: "Smartphones" },
    { id: "102", name: "iPad Air", price: 599, category: "Tablets" },
  ]);

  console.log("\nSearching: 'macbok' (Typo test)...");
  const results = await koski.search("products", "macbok");

  console.log("\nResults found:");
  console.log(JSON.stringify(results, null, 2));
}

if (require.main === module) {
  runExample().catch((/** @type {any} */ err) => {
    console.error("Example failed:", err.message);
  });
}

module.exports = KoskidexClient;
