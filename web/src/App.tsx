import { useEffect, useState, useCallback } from 'react';
import { Navbar } from './components/Navbar';
import { Sidebar } from './components/Sidebar';
import { PageView } from './components/PageView';
import { PageEditor } from './components/PageEditor';
import { RevisionHistory } from './components/RevisionHistory';
import { ApiModal } from './components/ApiModal';
import { AuthModal } from './components/AuthModal';
import { UserApiKeysModal } from './components/UserApiKeysModal';
import { AdminUsersModal } from './components/AdminUsersModal';
import { wikiAPI } from './api';
import type { Page, Revision, Tag, User, CreatePageInput, UpdatePageInput } from './types';

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
  
  // Auth & User state
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [showAuthModal, setShowAuthModal] = useState(false);
  const [showApiKeysModal, setShowApiKeysModal] = useState(false);
  const [showAdminUsersModal, setShowAdminUsersModal] = useState(false);

  const [showApiModal, setShowApiModal] = useState(false);
  const [healthOk, setHealthOk] = useState(true);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState('');

  // Check backend health & fetch logged-in user
  useEffect(() => {
    wikiAPI.getHealth()
      .then(() => setHealthOk(true))
      .catch(() => setHealthOk(false));

    wikiAPI.getMe()
      .then(setCurrentUser)
      .catch(() => setCurrentUser(null));
  }, []);

  // Dynamic Browser Tab Title
  useEffect(() => {
    if (viewMode === 'new') {
      document.title = 'Skapa ny sida | PTC Wiki';
    } else if (viewMode === 'edit' && activePage) {
      document.title = `Redigerar: ${activePage.title} | PTC Wiki`;
    } else if (viewMode === 'revisions' && activePage) {
      document.title = `Ändringshistorik: ${activePage.title} | PTC Wiki`;
    } else if (viewMode === 'view' && activePage) {
      document.title = `${activePage.title} | PTC Wiki`;
    } else {
      document.title = 'Pharatropic Wiki (PTC Wiki)';
    }
  }, [viewMode, activePage]);

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
      if (fetchedPages.length > 0 && !activeSlug) {
        setActiveSlug(fetchedPages[0].slug);
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Kunde inte ansluta till servern.');
    } finally {
      setIsLoading(false);
    }
  }, [searchQuery, activeTag, activeSlug]);

  useEffect(() => {
    loadSidebarData();
  }, [loadSidebarData]);

  // Fetch Active Page Details
  const loadPageDetails = useCallback(async (slug: string) => {
    setIsLoading(true);
    setErrorMsg('');
    try {
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
    if (window.confirm(`Är du säker på att du vill ta bort sidan "${activePage.title}"?`)) {
      try {
        await wikiAPI.deletePage(activeSlug);
        setActiveSlug(null);
        setActivePage(null);
        await loadSidebarData();
      } catch (err: any) {
        alert(err.message || 'Misslyckades att ta bort sidan.');
      }
    }
  };

  // Fetch Revisions
  const handleViewHistory = async () => {
    if (!activeSlug) return;
    try {
      const fetchedRevisions = await wikiAPI.getRevisions(activeSlug);
      setRevisions(fetchedRevisions);
      setViewMode('revisions');
    } catch (err: any) {
      alert(err.message || 'Kunde inte hämta ändringshistorik.');
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

  const handleLogout = async () => {
    await wikiAPI.logout();
    setCurrentUser(null);
  };

  return (
    <div className="app-container">
      <Navbar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onNewPage={() => {
          setNewTitlePrefill('');
          setViewMode('new');
        }}
        onOpenApiModal={() => setShowApiModal(true)}
        onHomeClick={() => {
          if (pages.length > 0) setActiveSlug(pages[0].slug);
          setViewMode('view');
        }}
        healthOk={healthOk}
        currentUser={currentUser}
        onOpenLogin={() => setShowAuthModal(true)}
        onOpenApiKeys={() => setShowApiKeysModal(true)}
        onOpenAdminUsers={() => setShowAdminUsersModal(true)}
        onLogout={handleLogout}
      />

      <main className="main-layout" style={{ maxWidth: '1400px', margin: '0 auto', width: '100%' }}>
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
            setActiveTag(activeTag === tagSlug ? null : tagSlug);
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

      {showAuthModal && (
        <AuthModal
          onClose={() => setShowAuthModal(false)}
          onSuccess={(user) => setCurrentUser(user)}
        />
      )}

      {showApiKeysModal && (
        <UserApiKeysModal
          onClose={() => setShowApiKeysModal(false)}
        />
      )}

      {showAdminUsersModal && (
        <AdminUsersModal
          onClose={() => setShowAdminUsersModal(false)}
        />
      )}
    </div>
  );
}

export default App;
