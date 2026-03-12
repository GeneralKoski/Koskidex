<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;

class KoskidexClient
{
    protected string $host;
    protected string $apiKey;

    public function __construct()
    {
        $this->host = config('koskidex.host');
        $this->apiKey = config('koskidex.api_key', '');
    }

    /**
     * Create a pre-configured HTTP request instance
     */
    protected function request()
    {
        $req = Http::acceptJson()->timeout(5);
        if ($this->apiKey) {
            $req->withToken($this->apiKey);
        }
        return $req;
    }

    /**
     * Create a new empty index
     */
    public function createIndex(string $name)
    {
        return $this->request()->post("{$this->host}/indexes", ['name' => $name])->json();
    }

    /**
     * Batch push documents into the index
     */
    public function addDocuments(string $index, array $documents)
    {
        return $this->request()->post("{$this->host}/indexes/{$index}/documents", $documents)->json();
    }

    /**
     * Remove a document from the index
     */
    public function deleteDocument(string $index, string $id)
    {
        return $this->request()->delete("{$this->host}/indexes/{$index}/documents/{$id}")->json();
    }
}
