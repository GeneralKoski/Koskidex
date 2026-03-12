import requests
import json

class KoskidexClient:
    def __init__(self, host='http://localhost:7700', api_key=None):
        self.host = host.rstrip('/')
        self.session = requests.Session()
        if api_key:
            self.session.headers.update({'Authorization': f'Bearer {api_key}'})

    def create_index(self, name: str):
        res = self.session.post(f"{self.host}/indexes", json={"name": name})
        res.raise_for_status()
        return res.json()

    def add_documents(self, index_name: str, documents: list):
        res = self.session.post(f"{self.host}/indexes/{index_name}/documents", json=documents)
        res.raise_for_status()
        return res.json()

    def add_document(self, index_name: str, document: dict):
        res = self.session.post(f"{self.host}/indexes/{index_name}/documents", json=document)
        res.raise_for_status()
        return res.json()

    def search(self, index_name: str, query: str):
        res = self.session.get(f"{self.host}/indexes/{index_name}/search", params={"q": query})
        res.raise_for_status()
        return res.json()

# ============== USAGE EXAMPLE ==============
if __name__ == "__main__":
    client = KoskidexClient()

    try:
        print("Creating 'articles' index...")
        client.create_index("articles")
    except Exception as e:
        print("Index might already exist.")

    print("Adding documents in batch...")
    client.add_documents("articles", [
        {"id": "a1", "title": "Understanding Python Decorators", "tags": "python, functions"},
        {"id": "a2", "title": "A Guide to Go Interfaces", "tags": "golang, interfaces"},
        {"id": "a3", "title": "Building REST APIs", "tags": "api, rest, web"}
    ])

    print("Searching: 'gude' (Typo test)...")
    results = client.search("articles", "gude")
    
    print("\nResults found:")
    print(json.dumps(results, indent=2))
