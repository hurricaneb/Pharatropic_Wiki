import { useEffect, useState, useCallback } from 'react';
import { Navbar } from './components/Navbar';
import { Sidebar } from './components/Sidebar';
import { PageView } from './components/PageView';
import { PageEditor } from './components/PageEditor';
import { RevisionHistory } from './components/RevisionHistory';
import { ApiModal } from './components/ApiModal';
import { wikiAPI } from './api';
import type { Page, Revision, Tag, CreatePageInput, UpdatePageInput } from './types';

export function App() {
  const [pages, setPages] = useState<Page[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [activeSlug, setActiveSlug] = useState<string | null>(null);
  const [activePage, setActivePage] = useState<Page | null>(null);
  const [revisions, setRevisions] = useState<Revision[]>([]);
  
  const [viewMode, setViewMode] = useState<'view' | 'edit' | 'new' | 'revisions'>('view');
  const [searchQuery, setSearchQuery] = useState('');
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const [newTitlePrefill, setNewTitlePrefill] = useState('');
  
  const [showApiModal, setShowApiModal] = useState(false);
  const [healthOk, setHealthOk] = useState(true);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState('');

  // Check backend health
  useEffect(() => {
    wikiAPI.getHealth()
      .then(() => setHealthOk(true))
      .catch(() => setHealthOk(false));
  }, []);

  // Fetch page list & tags
  const loadSidebarData = useCallback(async () => {
    try {
      const [fetchedPages, fetchedTags] = await Promise.all([
        wikiAPI.listPages(searchQuery, activeTag || undefined),
        wikiAPI.listTags(),
      ]);
      setPages(fetchedPages);
      setTags(fetchedTags);

      // Default select first page if none active
      if (!activeSlug && fetchedPages.length > 0) {
        setActiveSlug(fetchedPages[0].slug);
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Kunde inte ansluta till Go Wiki API:t.');
    } finally {
      setIsLoading(false);
    }
  }, [searchQuery, activeTag, activeSlug]);

  useEffect(() => {
    loadSidebarData();
  }, [loadSidebarData]);

  // Fetch active page details
  const loadPageDetails = useCallback(async (slug: string) => {
    try {
      setIsLoading(true);
      const pageData = await wikiAPI.getPage(slug);
      setActivePage(pageData);
    } catch (err: any) {
      setErrorMsg(err.message || 'Kunde inte hämta sidan.');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (activeSlug) {
      loadPageDetails(activeSlug);
    }
  }, [activeSlug, loadPageDetails]);

  // Handle Create Page
  const handleCreatePage = async (input: CreatePageInput) => {
    const newPage = await wikiAPI.createPage(input);
    setNewTitlePrefill('');
    await loadSidebarData();
    setActiveSlug(newPage.slug);
    setViewMode('view');
  };

  // Handle Update Page
  const handleUpdatePage = async (input: UpdatePageInput) => {
    if (!activeSlug) return;
    const updatedPage = await wikiAPI.updatePage(activeSlug, input);
    await loadSidebarData();
    setActiveSlug(updatedPage.slug);
    await loadPageDetails(updatedPage.slug);
    setViewMode('view');
  };

  // Handle Delete Page
  const handleDeletePage = async () => {
    if (!activeSlug || !activePage) return;
    if (!window.confirm(`Är du säker på att du vill radera sidan "${activePage.title}"?`)) {
      return;
    }

    try {
      await wikiAPI.deletePage(activeSlug);
      setActiveSlug(null);
      setActivePage(null);
      await loadSidebarData();
      setViewMode('view');
    } catch (err: any) {
      alert(err.message || 'Misslyckades att radera sidan.');
    }
  };

  // Load Revision History
  const handleViewHistory = async () => {
    if (!activeSlug) return;
    try {
      const revs = await wikiAPI.getRevisions(activeSlug);
      setRevisions(revs);
      setViewMode('revisions');
    } catch (err: any) {
      alert(err.message || 'Misslyckades att hämta revisioner.');
    }
  };

  // Handle Revert Revision
  const handleRevertRevision = async (revisionId: number) => {
    if (!activeSlug) return;
    const revertedPage = await wikiAPI.revertRevision(activeSlug, revisionId);
    await loadSidebarData();
    setActiveSlug(revertedPage.slug);
    await loadPageDetails(revertedPage.slug);
    setViewMode('view');
  };

  return (
    <div className="app-container">
      <Navbar
        searchQuery={searchQuery}
        onSearchChange={(q) => {
          setSearchQuery(q);
          if (viewMode !== 'view') setViewMode('view');
        }}
        onNewPage={() => setViewMode('new')}
        onOpenApiModal={() => setShowApiModal(true)}
        onHomeClick={() => {
          setActiveTag(null);
          setSearchQuery('');
          setViewMode('view');
        }}
        healthOk={healthOk}
      />

      <main className="main-layout">
        <Sidebar
          pages={pages}
          tags={tags}
          activeSlug={activeSlug}
          activeTag={activeTag}
          onSelectPage={(slug) => {
            setActiveSlug(slug);
            setViewMode('view');
          }}
          onSelectTag={(tagSlug) => {
            setActiveTag(tagSlug);
            if (viewMode !== 'view') setViewMode('view');
          }}
        />

        <section style={{ minWidth: 0 }}>
          {errorMsg && (
            <div style={{ padding: '16px', background: 'rgba(239, 68, 68, 0.15)', border: '1px solid rgba(239, 68, 68, 0.3)', borderRadius: 'var(--radius-md)', color: '#fca5a5', marginBottom: '20px' }}>
              {errorMsg}
            </div>
          )}

          {viewMode === 'new' && (
            <PageEditor
              initialTitle={newTitlePrefill}
              onSave={handleCreatePage as any}
              onCancel={() => {
                setNewTitlePrefill('');
                setViewMode('view');
              }}
            />
          )}

          {viewMode === 'edit' && activePage && (
            <PageEditor
              initialPage={activePage}
              onSave={handleUpdatePage as any}
              onCancel={() => setViewMode('view')}
            />
          )}

          {viewMode === 'revisions' && activePage && (
            <RevisionHistory
              page={activePage}
              revisions={revisions}
              onBack={() => setViewMode('view')}
              onRevertRevision={handleRevertRevision}
            />
          )}

          {viewMode === 'view' && activePage && (
            <PageView
              page={activePage}
              pages={pages}
              onEdit={() => setViewMode('edit')}
              onViewHistory={handleViewHistory}
              onDelete={handleDeletePage}
              onOpenApiModal={() => setShowApiModal(true)}
              onSelectPage={(slug) => {
                setActiveSlug(slug);
                setViewMode('view');
              }}
              onCreateMissingPage={(title) => {
                setNewTitlePrefill(title);
                setViewMode('new');
              }}
              onRefreshPage={() => activeSlug && loadPageDetails(activeSlug)}
            />
          )}

          {viewMode === 'view' && !activePage && !isLoading && (
            <div className="glass-panel" style={{ padding: '48px', textAlign: 'center' }}>
              <h2 style={{ fontSize: '1.5rem', marginBottom: '12px', color: '#fff' }}>Ingen wiki-sida vald</h2>
              <p style={{ color: 'var(--text-muted)', marginBottom: '20px' }}>
                Välj en sida i sidofältet till vänster eller skapa en ny wiki-sida.
              </p>
              <button className="btn btn-primary" onClick={() => setViewMode('new')}>
                Skapa Ny Sida
              </button>
            </div>
          )}
        </section>
      </main>

      {showApiModal && (
        <ApiModal
          page={activePage}
          onClose={() => setShowApiModal(false)}
        />
      )}
    </div>
  );
}

export default App;
