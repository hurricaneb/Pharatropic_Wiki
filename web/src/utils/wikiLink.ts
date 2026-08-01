/**
 * Slugifies a title string to match Go's slug.Make format
 */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/å/g, 'a')
    .replace(/ä/g, 'a')
    .replace(/ö/g, 'o')
    .replace(/é/g, 'e')
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-');
}

/**
 * Transforms [[Page Title]] and [[Page Title|Display Text]]
 * into standard markdown links with a custom #wikilink: scheme.
 * Example:
 * [[Välkommen till Wikin]] -> [Välkommen till Wikin](#wikilink:V%C3%A4lkommen%20till%20Wikin)
 * [[Projektarkitektur|Vår Arkitektur]] -> [Vår Arkitektur](#wikilink:Projektarkitektur)
 */
export function transformWikiLinks(markdown: string): string {
  if (!markdown) return '';

  // Match [[Title]] or [[Title|Label]]
  const wikiLinkRegex = /\[\[([^\]\|]+)(?:\|([^\]]+))?\]\]/g;

  return markdown.replace(wikiLinkRegex, (_, title, label) => {
    const cleanTitle = title.trim();
    const displayLabel = label ? label.trim() : cleanTitle;
    const encodedTitle = encodeURIComponent(cleanTitle);
    return `[${displayLabel}](#wikilink:${encodedTitle})`;
  });
}
