<?php

class KoskidexClient
{
    private string $host;
    private string $apiKey;

    public function __construct(string $host = 'http://localhost:7700', string $apiKey = '')
    {
        $this->host = rtrim($host, '/');
        $this->apiKey = $apiKey;
    }

    private function request(string $method, string $path, array $data = [])
    {
        $ch = curl_init($this->host . $path);

        $headers = ['Content-Type: application/json'];
        if ($this->apiKey) {
            $headers[] = 'Authorization: Bearer ' . $this->apiKey;
        }

        $options = [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_HTTPHEADER => $headers,
            CURLOPT_CUSTOMREQUEST => strtoupper($method),
        ];

        if ($method !== 'GET' && !empty($data)) {
            $options[CURLOPT_POSTFIELDS] = json_encode($data);
        }

        curl_setopt_array($ch, $options);
        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        $result = json_decode($response, true);
        
        if ($httpCode >= 400) {
            throw new Exception("Koskidex Error [{$httpCode}]: " . ($result['error'] ?? $response));
        }

        return $result;
    }

    public function createIndex(string $name)
    {
        return $this->request('POST', '/indexes', ['name' => $name]);
    }

    public function addDocuments(string $indexName, array $documents)
    {
        return $this->request('POST', "/indexes/{$indexName}/documents", $documents);
    }

    public function search(string $indexName, string $query)
    {
        return $this->request('GET', "/indexes/{$indexName}/search?q=" . urlencode($query));
    }
}

// ============== USAGE EXAMPLE =================

if (php_sapi_name() === 'cli' && basename(__FILE__) === basename($_SERVER['SCRIPT_FILENAME'])) {
    
    $koski = new KoskidexClient('http://localhost:7700');

    try {
        echo "Creating 'books' index...\n";
        $koski->createIndex('books');
    } catch (Exception $e) {
        echo "Index ignored (already exists or other error)\n";
    }

    echo "Adding books in batch...\n";
    $koski->addDocuments('books', [
        ['id' => 'b1', 'title' => 'The Lord of the Rings', 'author' => 'Tolkien'],
        ['id' => 'b2', 'title' => 'Harry Potter', 'author' => 'Rowling'],
        ['id' => 'b3', 'title' => '1984', 'author' => 'Orwell'],
    ]);

    echo "Searching for 'Hary' (Typo test)...\n";
    $result = $koski->search('books', 'Hary');

    echo "\nResults:\n";
    print_r($result);
}
