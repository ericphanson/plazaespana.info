# Icon Options for Plaza España Calendar

This document explores alternatives to emoji icons with permissively-licensed SVG icons that can be embedded inline in our HTML.

## Current Usage

Our site currently uses these emojis:
- 🎪 - Ongoing events section header
- 📍 - Distance indicator in event cards
- 📅 - Calendar/date-based time groups
- ⏰ - "Happening Now" time group
- 🎉 - "This Weekend" time group
- 🎭 - Cultural events (build report)
- 🎉 - City events (build report)

## Requirements

- **Permissive license** (MIT, Apache 2.0, or similar)
- **Inline SVG** - Must be embeddable directly in HTML (no external dependencies)
- **Small file size** - Keep total page size minimal
- **Accessibility** - Works without color alone
- **Consistent style** - Icons should feel cohesive as a set

---

## Option 1: Lucide Icons (MIT License)

**Website:** https://lucide.dev
**License:** MIT (attribution not required)
**Style:** Clean, consistent line icons with 24×24 viewBox
**File Size:** ~200-400 bytes per icon

### Visual Examples

#### Tent (for ongoing events 🎪)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <path d="M3.5 21 14 3"/>
  <path d="M20.5 21 10 3"/>
  <path d="M15.5 21 12 15l-3.5 6"/>
  <path d="M2 21h20"/>
</svg>
```

#### Map Pin (for distance 📍)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/>
  <circle cx="12" cy="10" r="3"/>
</svg>
```

#### Calendar (for date groups 📅)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <rect width="18" height="18" x="3" y="4" rx="2" ry="2"/>
  <line x1="16" x2="16" y1="2" y2="6"/>
  <line x1="8" x2="8" y1="2" y2="6"/>
  <line x1="3" x2="21" y1="10" y2="10"/>
</svg>
```

#### Clock (for "happening now" ⏰)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <circle cx="12" cy="12" r="10"/>
  <polyline points="12 6 12 12 16 14"/>
</svg>
```

#### Party Popper (for weekend/celebration 🎉)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <path d="M5.8 11.3 2 22l10.7-3.79"/>
  <path d="M4 3h.01"/>
  <path d="M22 8h.01"/>
  <path d="M15 2h.01"/>
  <path d="M22 20h.01"/>
  <circle cx="12" cy="12" r="2"/>
  <path d="m13.4 10.6 6.3-6.3"/>
  <path d="m10.6 13.4-6.3 6.3"/>
</svg>
```

#### Theater/Drama Mask (for cultural events 🎭)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <path d="M2 16.1A5 5 0 0 1 5.9 20M2 12.05A9 9 0 0 1 9.95 20M2 8V6a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v2"/>
  <path d="M2 14.66a9 9 0 0 0 7.34 0"/>
</svg>
```

### Pros
- Very clean, modern aesthetic
- Excellent consistency across icons
- Small file sizes
- Works great with `currentColor` for easy theming
- No attribution required

### Cons
- Line-only style might feel less playful than emojis
- May need color fills to add personality

---

## Option 2: Phosphor Icons (MIT License)

**Website:** https://phosphoricons.com
**License:** MIT (attribution not required)
**Style:** Flexible, available in multiple weights (thin/light/regular/bold/fill/duotone)
**File Size:** ~300-600 bytes per icon

### Visual Examples (using "fill" weight for more personality)

#### Tent (for ongoing events 🎪)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 256 256" fill="currentColor">
  <path d="M232,208H224V104a16,16,0,0,0-16-16H48a16,16,0,0,0-16,16V208H24a8,8,0,0,0,0,16H232a8,8,0,0,0,0-16ZM128,104v46.9L104.5,184H88V104ZM48,104H72v80H56a12,12,0,0,0-8,3.08ZM208,187.08a12,12,0,0,0-8-3.08H184V104h24Z"/>
</svg>
```

#### Map Pin (for distance 📍)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 256 256" fill="currentColor">
  <path d="M128,64a40,40,0,1,0,40,40A40,40,0,0,0,128,64Zm0,64a24,24,0,1,1,24-24A24,24,0,0,1,128,128Zm0-112a88.1,88.1,0,0,0-88,88c0,31.4,14.51,64.68,42,96.25a254.19,254.19,0,0,0,41.45,38.3,8,8,0,0,0,9.18,0A254.19,254.19,0,0,0,174,200.25c27.45-31.57,42-64.85,42-96.25A88.1,88.1,0,0,0,128,16Zm0,206c-16.53-13-72-60.75-72-118a72,72,0,0,1,144,0C200,161.23,144.53,209,128,222Z"/>
</svg>
```

#### Calendar (for date groups 📅)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 256 256" fill="currentColor">
  <path d="M208,32H184V24a8,8,0,0,0-16,0v8H88V24a8,8,0,0,0-16,0v8H48A16,16,0,0,0,32,48V208a16,16,0,0,0,16,16H208a16,16,0,0,0,16-16V48A16,16,0,0,0,208,32ZM72,48v8a8,8,0,0,0,16,0V48h80v8a8,8,0,0,0,16,0V48h24V80H48V48ZM208,208H48V96H208V208Z"/>
</svg>
```

#### Clock (for "happening now" ⏰)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 256 256" fill="currentColor">
  <path d="M128,24A104,104,0,1,0,232,128,104.11,104.11,0,0,0,128,24Zm0,192a88,88,0,1,1,88-88A88.1,88.1,0,0,1,128,216Zm64-88a8,8,0,0,1-8,8H128a8,8,0,0,1-8-8V72a8,8,0,0,1,16,0v48h48A8,8,0,0,1,192,128Z"/>
</svg>
```

#### Confetti (for weekend/celebration 🎉)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 256 256" fill="currentColor">
  <path d="M111.49,52.63a15.8,15.8,0,0,0-26,5.77L33,202.78A15.83,15.83,0,0,0,47.76,224a16,16,0,0,0,5.46-1l144.37-52.5a15.8,15.8,0,0,0,5.78-26Zm-8.33,135.21-35-35,13.16-36.21,58.05,58.05Zm-55,20L64,168.1l15.11,15.11ZM192,152.6,103.4,64l27-27.62L192,128Z"/>
  <path d="M144,40a8,8,0,0,1,8-8h16a8,8,0,0,1,0,16H152A8,8,0,0,1,144,40Zm64,72a8,8,0,0,1,8,8v16a8,8,0,0,1-16,0V120A8,8,0,0,1,208,112ZM232,64a8,8,0,0,1-8,8h-8v8a8,8,0,0,1-16,0V72h-8a8,8,0,0,1,0-16h8V48a8,8,0,0,1,16,0v8h8A8,8,0,0,1,232,64ZM184,168h-8a8,8,0,0,0,0,16h8a8,8,0,0,0,0-16Z"/>
</svg>
```

#### Theater Masks (for cultural events 🎭)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 256 256" fill="currentColor">
  <path d="M216,40H40A16,16,0,0,0,24,56V96a104,104,0,0,0,208,0V56A16,16,0,0,0,216,40ZM96,120a8,8,0,1,1,8-8A8,8,0,0,1,96,120Zm64,0a8,8,0,1,1,8-8A8,8,0,0,1,160,120Zm56-24a88.1,88.1,0,0,1-176,0V56H216Z"/>
  <path d="M128,152a39.94,39.94,0,0,0-33.93,19,8,8,0,0,0,13.86,8,24,24,0,0,1,40.14,0,8,8,0,1,0,13.86-8A39.94,39.94,0,0,0,128,152Z"/>
</svg>
```

### Pros
- Multiple weight options allow for flexibility
- Fill variants add visual interest and personality
- Duotone option available for two-color designs
- Very comprehensive icon set
- Small file sizes

### Cons
- Fill variants can be slightly larger than line icons
- Might be visually heavier than emojis depending on context

---

## Option 3: Bootstrap Icons (MIT License)

**Website:** https://icons.getbootstrap.com
**License:** MIT (attribution not required)
**Style:** Friendly, slightly rounded, available in both outline and fill
**File Size:** ~200-500 bytes per icon

### Visual Examples

#### Broadcast (for ongoing events 🎪)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
  <path d="M3.05 3.05a7 7 0 0 0 0 9.9.5.5 0 0 1-.707.707 8 8 0 0 1 0-11.314.5.5 0 0 1 .707.707zm2.122 2.122a4 4 0 0 0 0 5.656.5.5 0 1 1-.708.708 5 5 0 0 1 0-7.072.5.5 0 0 1 .708.708zm5.656-.708a.5.5 0 0 1 .708 0 5 5 0 0 1 0 7.072.5.5 0 1 1-.708-.708 4 4 0 0 0 0-5.656.5.5 0 0 1 0-.708zm2.122-2.12a.5.5 0 0 1 .707 0 8 8 0 0 1 0 11.313.5.5 0 0 1-.707-.707 7 7 0 0 0 0-9.9.5.5 0 0 1 0-.707zM10 8a2 2 0 1 1-4 0 2 2 0 0 1 4 0z"/>
</svg>
```

#### Geo Alt Fill (for distance 📍)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
  <path d="M8 16s6-5.686 6-10A6 6 0 0 0 2 6c0 4.314 6 10 6 10zm0-7a3 3 0 1 1 0-6 3 3 0 0 1 0 6z"/>
</svg>
```

#### Calendar Event (for date groups 📅)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
  <path d="M11 6.5a.5.5 0 0 1 .5-.5h1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-1a.5.5 0 0 1-.5-.5v-1z"/>
  <path d="M3.5 0a.5.5 0 0 1 .5.5V1h8V.5a.5.5 0 0 1 1 0V1h1a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2V3a2 2 0 0 1 2-2h1V.5a.5.5 0 0 1 .5-.5zM1 4v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V4H1z"/>
</svg>
```

#### Clock Fill (for "happening now" ⏰)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
  <path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zM8 3.5a.5.5 0 0 0-1 0V9a.5.5 0 0 0 .252.434l3.5 2a.5.5 0 0 0 .496-.868L8 8.71V3.5z"/>
</svg>
```

#### Party Horn (for weekend/celebration 🎉)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
  <path d="m13.106 7.222 2.97-2.97A1.5 1.5 0 0 0 14.95 2.12l-2.97 2.97L5.638 3.67l1.444 1.444L13.106 7.222zm-.736 4.45-1.392-1.392L4.536 17l2.168-2.168 5.666-4.159zm-3.123-2.95-4.95-4.95a1.5 1.5 0 0 0-2.121 2.121L7.126 10.9l2.121-2.121zM2 12.414a1.414 1.414 0 1 0 0 2.828 1.414 1.414 0 0 0 0-2.828z"/>
</svg>
```

#### Masks Theater (for cultural events 🎭)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
  <path d="M0 4a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H1a1 1 0 0 1-1-1V4zm3.5 5.5a.5.5 0 1 0 0-1 .5.5 0 0 0 0 1zm9 0a.5.5 0 1 0 0-1 .5.5 0 0 0 0 1zM5 6a1 1 0 1 0 0 2 1 1 0 0 0 0-2zm6 0a1 1 0 1 0 0 2 1 1 0 0 0 0-2zM8 6c-.646 0-1.278.285-1.67.765a.5.5 0 0 0 .74.673A1.238 1.238 0 0 1 8 7c.345 0 .678.143.93.438a.5.5 0 0 0 .74-.673A2.238 2.238 0 0 0 8 6z"/>
</svg>
```

### Pros
- Friendly, approachable style
- Good balance between playful and professional
- Very popular/widely recognized
- Comprehensive set with many variants
- Fill versions add visual weight

### Cons
- 16×16 viewBox might require more scaling adjustments
- Style is somewhat generic (very widely used)

---

## Option 4: Heroicons (MIT License) - Bonus Option

**Website:** https://heroicons.com
**License:** MIT by Tailwind Labs
**Style:** Clean, modern, available in outline/solid with 24×24 viewBox
**File Size:** ~150-400 bytes per icon

### Visual Examples (using solid variants)

#### Megaphone (for ongoing events 🎪)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
  <path d="M16.881 4.345A23.112 23.112 0 018.25 6H4.5a2.25 2.25 0 00-2.25 2.25v4.5A2.25 2.25 0 004.5 15h3.75a23.112 23.112 0 018.631 1.655 1.5 1.5 0 002.419-1.179V5.524a1.5 1.5 0 00-2.419-1.179zM19.5 12c0 .414-.168.791-.44 1.06a1.5 1.5 0 001.44 2.44A4.5 4.5 0 0021.75 12a4.5 4.5 0 00-1.25-3.5 1.5 1.5 0 00-1.44 2.44c.272.269.44.646.44 1.06zM10.5 18.75a.75.75 0 000 1.5h1.5a.75.75 0 000-1.5h-1.5z"/>
  <path d="M10.5 21.75a.75.75 0 000 1.5h1.5a.75.75 0 000-1.5h-1.5z"/>
</svg>
```

#### Map Pin (for distance 📍)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
  <path fill-rule="evenodd" d="M11.54 22.351l.07.04.028.016a.76.76 0 00.723 0l.028-.015.071-.041a16.975 16.975 0 001.144-.742 19.58 19.58 0 002.683-2.282c1.944-1.99 3.963-4.98 3.963-8.827a8.25 8.25 0 00-16.5 0c0 3.846 2.02 6.837 3.963 8.827a19.58 19.58 0 002.682 2.282 16.975 16.975 0 001.145.742zM12 13.5a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd"/>
</svg>
```

#### Calendar (for date groups 📅)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
  <path d="M12.75 12.75a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM7.5 15.75a.75.75 0 100-1.5.75.75 0 000 1.5zM8.25 17.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM9.75 15.75a.75.75 0 100-1.5.75.75 0 000 1.5zM10.5 17.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM12 15.75a.75.75 0 100-1.5.75.75 0 000 1.5zM12.75 17.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM14.25 15.75a.75.75 0 100-1.5.75.75 0 000 1.5zM15 17.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM16.5 15.75a.75.75 0 100-1.5.75.75 0 000 1.5zM15 12.75a.75.75 0 11-1.5 0 .75.75 0 011.5 0zM16.5 13.5a.75.75 0 100-1.5.75.75 0 000 1.5z"/>
  <path fill-rule="evenodd" d="M6.75 2.25A.75.75 0 017.5 3v1.5h9V3A.75.75 0 0118 3v1.5h.75a3 3 0 013 3v11.25a3 3 0 01-3 3H5.25a3 3 0 01-3-3V7.5a3 3 0 013-3H6V3a.75.75 0 01.75-.75zm13.5 9a1.5 1.5 0 00-1.5-1.5H5.25a1.5 1.5 0 00-1.5 1.5v7.5a1.5 1.5 0 001.5 1.5h13.5a1.5 1.5 0 001.5-1.5v-7.5z" clip-rule="evenodd"/>
</svg>
```

#### Clock (for "happening now" ⏰)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
  <path fill-rule="evenodd" d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25zM12.75 6a.75.75 0 00-1.5 0v6c0 .414.336.75.75.75h4.5a.75.75 0 000-1.5h-3.75V6z" clip-rule="evenodd"/>
</svg>
```

#### Sparkles (for weekend/celebration 🎉)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
  <path fill-rule="evenodd" d="M9 4.5a.75.75 0 01.721.544l.813 2.846a3.75 3.75 0 002.576 2.576l2.846.813a.75.75 0 010 1.442l-2.846.813a3.75 3.75 0 00-2.576 2.576l-.813 2.846a.75.75 0 01-1.442 0l-.813-2.846a3.75 3.75 0 00-2.576-2.576l-2.846-.813a.75.75 0 010-1.442l2.846-.813A3.75 3.75 0 007.466 7.89l.813-2.846A.75.75 0 019 4.5zM18 1.5a.75.75 0 01.728.568l.258 1.036c.236.94.97 1.674 1.91 1.91l1.036.258a.75.75 0 010 1.456l-1.036.258c-.94.236-1.674.97-1.91 1.91l-.258 1.036a.75.75 0 01-1.456 0l-.258-1.036a2.625 2.625 0 00-1.91-1.91l-1.036-.258a.75.75 0 010-1.456l1.036-.258a2.625 2.625 0 001.91-1.91l.258-1.036A.75.75 0 0118 1.5zM16.5 15a.75.75 0 01.712.513l.394 1.183c.15.447.5.799.948.948l1.183.395a.75.75 0 010 1.422l-1.183.395c-.447.15-.799.5-.948.948l-.395 1.183a.75.75 0 01-1.422 0l-.395-1.183a1.5 1.5 0 00-.948-.948l-1.183-.395a.75.75 0 010-1.422l1.183-.395c.447-.15.799-.5.948-.948l.395-1.183A.75.75 0 0116.5 15z" clip-rule="evenodd"/>
</svg>
```

#### Film (for cultural events 🎭)
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
  <path d="M4.5 4.5a3 3 0 00-3 3v9a3 3 0 003 3h8.25a3 3 0 003-3v-9a3 3 0 00-3-3H4.5zM19.94 18.75l-2.69-2.69V7.94l2.69-2.69c.944-.945 2.56-.276 2.56 1.06v11.38c0 1.336-1.616 2.005-2.56 1.06z"/>
</svg>
```

### Pros
- Very clean, professional aesthetic
- Excellent for modern, minimal designs
- Small file sizes
- Great consistency
- Designed by Tailwind team (quality assurance)

### Cons
- Might be too minimal/corporate for a fun events site
- Less playful than emojis

---

## Recommendation Matrix

| Use Case | Lucide | Phosphor | Bootstrap | Heroicons |
|----------|--------|----------|-----------|-----------|
| **Playfulness** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| **Professionalism** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **File Size** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Variety** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Theming** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |

---

## Implementation Notes

### Inline SVG Best Practices

1. **Use `currentColor` for stroke/fill** - Allows CSS color theming
2. **Set explicit width/height** - Ensures consistent sizing
3. **Add `aria-hidden="true"`** - Icons are decorative, not informational
4. **Keep viewBox** - Allows proper scaling

Example implementation:
```html
<span class="section-icon" aria-hidden="true">
  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <!-- SVG paths here -->
  </svg>
</span>
```

### CSS Styling
```css
.section-icon svg {
  display: inline-block;
  vertical-align: middle;
  margin-right: 0.5em;
}

/* Color variants for different contexts */
.cultural-badge svg { color: #9b59b6; }
.city-badge svg { color: #e67e22; }
```

### File Size Impact

Current emojis: ~0 bytes (unicode)
Estimated SVG cost: ~200-400 bytes per unique icon × ~6 icons = **~1.5-2.5 KB total**

Given our CSS is only 1.2 KB, adding inline SVGs would approximately double our asset payload but still keep total page size very small.

---

## My Recommendation: **Phosphor Icons (Fill Weight)**

**Reasoning:**
1. **Best balance** - Professional enough for information architecture, playful enough for event listings
2. **Visual weight** - Fill variants create stronger visual hierarchy than line-only icons
3. **Flexibility** - Can mix weights (fill for headers, regular for inline icons)
4. **Comprehensive** - Has all the icons we need with good semantic matches
5. **Small size** - Despite fill variants, still under 500 bytes per icon

**Alternative pick:** Bootstrap Icons if we want a friendlier, more rounded aesthetic that feels closer to emoji personality.

**Avoid:** Heroicons - too corporate/minimal for an events calendar site.
