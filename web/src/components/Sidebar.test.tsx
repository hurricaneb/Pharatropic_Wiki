import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Sidebar } from './Sidebar';
import type { Page, Tag } from '../types';

function makePage(overrides: Partial<Page> & { id: number; slug: string; title: string }): Page {
  return {
    summary: '',
    content: '',
    is_public: true,
    views: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

const noop = () => {};

describe('Sidebar', () => {
  it('shows an empty state when there are no pages', () => {
    render(<Sidebar pages={[]} tags={[]} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
    expect(screen.getByText('Inga sidor hittades.')).toBeInTheDocument();
  });

  it('renders top-level pages without indentation markers', () => {
    const pages = [makePage({ id: 1, slug: 'a', title: 'Page A' }), makePage({ id: 2, slug: 'b', title: 'Page B' })];
    render(<Sidebar pages={pages} tags={[]} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
    expect(screen.getByText('Page A')).toBeInTheDocument();
    expect(screen.getByText('Page B')).toBeInTheDocument();
  });

  it('hides a page with subpages until it becomes the active page', () => {
    const pages = [
      makePage({ id: 1, slug: 'parent', title: 'Parent' }),
      makePage({ id: 2, slug: 'child', title: 'Child', parent_id: 1 }),
    ];
    const { rerender } = render(
      <Sidebar pages={pages} tags={[]} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={noop} />
    );
    expect(screen.getByText('Parent')).toBeInTheDocument();
    expect(screen.queryByText('Child')).not.toBeInTheDocument();

    rerender(<Sidebar pages={pages} tags={[]} activeSlug="parent" activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
    expect(screen.getByText('Child')).toBeInTheDocument();
  });

  it('keeps subpages visible when a child itself is the active page', () => {
    const pages = [
      makePage({ id: 1, slug: 'parent', title: 'Parent' }),
      makePage({ id: 2, slug: 'child', title: 'Child', parent_id: 1 }),
    ];
    render(<Sidebar pages={pages} tags={[]} activeSlug="child" activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
    expect(screen.getByText('Child')).toBeInTheDocument();
  });

  it('renders an orphaned child (parent filtered out) as a top-level entry', () => {
    // Simulates a search/tag filter that excludes the parent but not the child
    const pages = [makePage({ id: 2, slug: 'child', title: 'Orphan Child', parent_id: 99 })];
    render(<Sidebar pages={pages} tags={[]} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
    expect(screen.getByText('Orphan Child')).toBeInTheDocument();
  });

  it('shows a lock icon for private pages', () => {
    const pages = [makePage({ id: 1, slug: 'a', title: 'Private Page', is_public: false })];
    render(<Sidebar pages={pages} tags={[]} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
    expect(screen.getByTitle('Privat sida (kräver inloggning)')).toBeInTheDocument();
  });

  it('calls onSelectPage with the clicked page slug', async () => {
    const user = userEvent.setup();
    const onSelectPage = vi.fn();
    const pages = [makePage({ id: 1, slug: 'a', title: 'Page A' })];
    render(<Sidebar pages={pages} tags={[]} activeSlug={null} activeTag={null} onSelectPage={onSelectPage} onSelectTag={noop} />);

    await user.click(screen.getByText('Page A'));
    expect(onSelectPage).toHaveBeenCalledWith('a');
  });

  it('shows the view count for each page', () => {
    const pages = [makePage({ id: 1, slug: 'a', title: 'Page A', views: 42 })];
    render(<Sidebar pages={pages} tags={[]} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('shows the total page count in the section header', () => {
    const pages = [makePage({ id: 1, slug: 'a', title: 'A' }), makePage({ id: 2, slug: 'b', title: 'B' })];
    render(<Sidebar pages={pages} tags={[]} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
    expect(screen.getByText('Alla Sidor (2)')).toBeInTheDocument();
  });

  describe('tags section', () => {
    const tags: Tag[] = [
      { id: 1, name: 'Go', slug: 'go' },
      { id: 2, name: 'React', slug: 'react' },
    ];

    it('shows a placeholder when there are no tags', () => {
      render(<Sidebar pages={[]} tags={[]} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
      expect(screen.getByText('Inga taggar än.')).toBeInTheDocument();
    });

    it('renders every tag', () => {
      render(<Sidebar pages={[]} tags={tags} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={noop} />);
      expect(screen.getByText('#Go')).toBeInTheDocument();
      expect(screen.getByText('#React')).toBeInTheDocument();
    });

    it('selects a tag on click', async () => {
      const user = userEvent.setup();
      const onSelectTag = vi.fn();
      render(<Sidebar pages={[]} tags={tags} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={onSelectTag} />);
      await user.click(screen.getByText('#Go'));
      expect(onSelectTag).toHaveBeenCalledWith('go');
    });

    it('deselects the active tag when clicked again', async () => {
      const user = userEvent.setup();
      const onSelectTag = vi.fn();
      render(<Sidebar pages={[]} tags={tags} activeSlug={null} activeTag="go" onSelectPage={noop} onSelectTag={onSelectTag} />);
      await user.click(screen.getByText('#Go'));
      expect(onSelectTag).toHaveBeenCalledWith(null);
    });

    it('shows a "Rensa" (clear) button only when a tag is active, and it clears the filter', async () => {
      const user = userEvent.setup();
      const onSelectTag = vi.fn();
      const { rerender } = render(
        <Sidebar pages={[]} tags={tags} activeSlug={null} activeTag={null} onSelectPage={noop} onSelectTag={onSelectTag} />
      );
      expect(screen.queryByText('Rensa')).not.toBeInTheDocument();

      rerender(<Sidebar pages={[]} tags={tags} activeSlug={null} activeTag="go" onSelectPage={noop} onSelectTag={onSelectTag} />);
      const clearBtn = screen.getByText('Rensa');
      await user.click(clearBtn);
      expect(onSelectTag).toHaveBeenCalledWith(null);
    });
  });
});
