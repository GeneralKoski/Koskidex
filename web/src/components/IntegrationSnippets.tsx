import { useState } from 'react';
import { Code, Terminal } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export default function IntegrationSnippets() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('curl');

  const tabs = [
    { id: 'curl', label: 'cURL' },
    { id: 'php', label: 'PHP (Laravel)' },
    { id: 'node', label: 'Node.js' },
    { id: 'python', label: 'Python' }
  ];

  const codeSnippets: Record<string, string> = {
    curl: `# 1. Create Index
curl -X POST http://localhost:7700/indexes -d '{"name": "movies"}'

# 2. Add Documents
curl -X POST http://localhost:7700/indexes/movies/documents \\
  -H 'Content-Type: application/json' -d '[
  {"id": "1", "title": "The Matrix", "genre": "Sci-Fi"}
]'

# 3. Search (Typo tolerant!)
curl "http://localhost:7700/indexes/movies/search?q=matrx"`,

    php: `// Using our Laravel Searchable Trait - No manual syncs!
use App\\Traits\\Searchable;

class Movie extends Model {
    use Searchable;
}

// Just save to MySQL, it seamlessly upserts to Koskidex
Movie::create(['title' => 'The Matrix']);

// Later, search with extreme speed matching typos!
$results = Movie::koskidexSearch('matrx');`,

    node: `const KoskidexClient = require('./koskidex_client');
const client = new KoskidexClient('http://localhost:7700');

// Bulk ingest 10,000 documents instantly
await client.addDocuments('movies', moviesArray);

// User made a typo "matrihx"? Koskidex finds it.
const results = await client.search('movies', 'matrihx');

console.log(results.hits);`,

    python: `from koskidex_client import KoskidexClient
client = KoskidexClient('http://localhost:7700')

# Create index
client.create_index('movies')

# Add documents
client.add_documents('movies', [
    {"id": "1", "title": "The Matrix"}
])

# Perform blazing fast search
res = client.search('movies', 'mtrx')`
  };

  return (
    <section id="integration" className="py-20 border-t border-slate-800/50 relative">
      <div className="container mx-auto px-4 max-w-4xl">
        <div className="text-center mb-12">
          <Code className="w-12 h-12 text-blue-500 mx-auto mb-4 opacity-80" />
          <h2 className="text-3xl font-bold mb-4">{t('integration.title')}</h2>
          <p className="text-slate-400 text-lg">{t('integration.subtitle')}</p>
        </div>

        <div className="glass-effect rounded-xl overflow-hidden shadow-2xl">
          <div className="flex border-b border-slate-800 bg-slate-900/80 overflow-x-auto hide-scrollbar">
            {tabs.map(tab => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                aria-label={tab.label}
                aria-selected={activeTab === tab.id}
                className={`px-6 py-4 text-sm font-semibold transition-all whitespace-nowrap border-b-2 ${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-400 bg-blue-500/5'
                    : 'border-transparent text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
                }`}
              >
                {tab.id === 'curl' && <Terminal className="w-4 h-4 inline-block mr-2 -mt-1 opacity-70" />}
                {tab.label}
              </button>
            ))}
          </div>

          <div className="p-6 bg-[#0B1120] relative group">
            <div className="absolute right-4 top-4 flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
              <button 
                onClick={() => {
                  navigator.clipboard.writeText(codeSnippets[activeTab]);
                }}
                aria-label={t('common.copy')}
                className="text-xs bg-slate-800 hover:bg-slate-700 text-slate-300 py-1.5 px-3 rounded-md font-medium transition-colors border border-slate-700/50"
              >
                {t('common.copy')}
              </button>
              <span className="text-xs bg-slate-800 text-slate-400 py-1.5 px-3 rounded-md font-mono border border-slate-700/50">{activeTab}</span>
            </div>
            <pre className="font-mono text-sm text-slate-300 overflow-x-auto leading-relaxed">
              <code>{codeSnippets[activeTab]}</code>
            </pre>
          </div>
        </div>
      </div>
    </section>
  );
}
