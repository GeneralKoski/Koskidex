<?php

/**
 * Standalone test script for Koskidex Laravel Client.
 * 
 * Usage:
 * 1. Ensure Koskidex is running on port 7700.
 * 2. Run: php test-client.php
 */

if (file_exists(__DIR__ . '/vendor/autoload.php')) {
    require_once __DIR__ . '/vendor/autoload.php';
}

// Mocking config() helper if not in a Laravel environment
if (!function_exists('config')) {
    function config($key, $default = null) {
        $configs = [
            'koskidex.host' => 'http://localhost:7700',
            'koskidex.api_key' => ''
        ];
        return $configs[$key] ?? $default;
    }
}

// Minimal manual inclusion for standalone test if vendor/autoload is missing
// In a real app, this is handled by Composer.
require_once __DIR__ . '/app/Services/KoskidexClient.php';

use App\Services\KoskidexClient;

echo "🚀 Starting Koskidex Laravel Client test...\n";

$client = new KoskidexClient();

try {
    echo "Creating 'test_index'...\n";
    $res = $client->createIndex('test_index');
    echo "Result: " . json_encode($res) . "\n\n";
    
    echo "Adding documents...\n";
    $docs = [
        ['id' => 'l1', 'title' => 'Laravel Guide', 'author' => 'Taylor Otwell'],
        ['id' => 'l2', 'title' => 'Go for Beginners', 'author' => 'Rob Pike']
    ];
    $res = $client->addDocuments('test_index', $docs);
    echo "Result: " . json_encode($res) . "\n\n";
    
    echo "Searching for 'laravl' (typo test)...\n";
    $res = $client->search('test_index', 'laravl');
    echo "Results found: " . count($res['hits'] ?? []) . "\n";
    echo json_encode($res, JSON_PRETTY_PRINT) . "\n";
    
    echo "\n✅ Test completed successfully!\n";
} catch (\Exception $e) {
    echo "❌ Error: " . $e->getMessage() . "\n";
}
