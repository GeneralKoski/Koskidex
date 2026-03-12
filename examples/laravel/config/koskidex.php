<?php

return [
    /*
    |--------------------------------------------------------------------------
    | Koskidex Connection URL
    |--------------------------------------------------------------------------
    | The host where your koskidex container is running.
    */
    'host' => env('KOSKIDEX_HOST', 'http://localhost:7700'),
    
    /*
    |--------------------------------------------------------------------------
    | API Key
    |--------------------------------------------------------------------------
    | Your secret key, if you started koskidex with the --api-key flag
    */
    'api_key' => env('KOSKIDEX_API_KEY', ''),
    
    /*
    |--------------------------------------------------------------------------
    | Indices Configuration and Model Mapping
    |--------------------------------------------------------------------------
    | Automatically map your Laravel Models (e.g. User, Location) 
    | to the corresponding index on Koskidex, specifying which fields to sync.
    */
    'indices' => [
        App\Models\User::class => [
            'index_name' => 'users',
            
            // Fields to send to Koskidex (including nullables like email)
            'searchable_fields' => ['name', 'surname', 'email'],
            
            // Expected hit threshold percentage to consider a match valid
            'hit_threshold' => 70, 

            // Optional: Index-specific settings inside Koskidex (e.g. stop words)
            'settings' => [
                'stop_words' => ['the', 'is', 'a'],
            ],
        ],

        // Example for Location model:
        // App\Models\Location::class => [
        //     'index_name' => 'locations',
        //     'searchable_fields' => ['address', 'city', 'zip', 'country_code'],
        // ],
    ],
];
