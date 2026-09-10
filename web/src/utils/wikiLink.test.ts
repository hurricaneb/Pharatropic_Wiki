import { describe, it, expect } from 'vitest';
import { slugify, transformWikiLinks } from './wikiLink';

describe('slugify', () => {
  it('lowercases and hyphenates spaces', () => {
    expect(slugify('Hello World')).toBe('hello-world');
  });

  it('transliterates Swedish characters', () => {
    expect(slugify('Åäö Guide')).toBe('aao-guide');
  });

  it('strips punctuation', () => {
    expect(slugify('API: The Guide!')).toBe('api-the-guide');
  });

  it('collapses multiple hyphens', () => {
    expect(slugify('a   b---c')).toBe('a-b-c');
  });

  it('trims leading/trailing whitespace before slugifying', () => {
    expect(slugify('  Padded Title  ')).toBe('padded-title');
  });

  it('returns an empty string for input with no valid characters', () => {
    expect(slugify('!!!')).toBe('');
  });
});

describe('transformWikiLinks', () => {
  it('returns an empty string for empty input', () => {
    expect(transformWikiLinks('')).toBe('');
  });

  it('leaves plain text without wikilinks unchanged', () => {
    expect(transformWikiLinks('just some text')).toBe('just some text');
  });

  it('converts a simple [[Title]] link', () => {
    expect(transformWikiLinks('See [[Some Page]] for details')).toBe(
      'See [Some Page](#wikilink:Some%20Page) for details'
    );
  });

  it('converts a [[Title|Label]] link using the label as display text', () => {
    expect(transformWikiLinks('[[Real Title|Click Here]]')).toBe(
      '[Click Here](#wikilink:Real%20Title)'
    );
  });

  it('handles multiple links in the same string', () => {
    const input = 'Go to [[Page One]] or [[Page Two|here]].';
    const output = transformWikiLinks(input);
    expect(output).toContain('[Page One](#wikilink:Page%20One)');
    expect(output).toContain('[here](#wikilink:Page%20Two)');
  });

  it('trims whitespace inside the brackets', () => {
    expect(transformWikiLinks('[[  Spaced Title  ]]')).toBe(
      '[Spaced Title](#wikilink:Spaced%20Title)'
    );
  });

  it('URL-encodes special characters in the title', () => {
    expect(transformWikiLinks('[[Åäö Page]]')).toContain('#wikilink:%C3%85%C3%A4%C3%B6%20Page');
  });

  it('does not touch normal markdown links', () => {
    const input = '[Normal Link](https://example.com)';
    expect(transformWikiLinks(input)).toBe(input);
  });
});
