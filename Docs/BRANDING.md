# JAX Trading Assistant - Branding Guide

This document outlines the branding guidelines, color palette, logo usage, and favicon management for the JAX Trading Assistant project.

## Logo

### Primary Logo

**Location**: [frontend/src/images/jax_ai_trader.svg](../frontend/src/images/jax_ai_trader.svg)

**Format**: SVG (Scalable Vector Graphics)

**Usage**:

- Primary branding element across the application
- Used in the header/navigation bar
- Used as the base for all favicon generation
- Scalable without quality loss

### Logo Guidelines

**Recommended Sizes**:

- **Header/Navigation**: 32-48px height
- **Sidebar**: 24-32px height
- **Large displays**: 60-80px height

**Spacing**:

- Maintain minimum 8px clear space around the logo
- Avoid crowding with text or other elements

**Color Variations**:

- Use the logo as-is on light backgrounds
- Ensure sufficient contrast for accessibility

**Don'ts**:

- Don't distort or stretch the logo
- Don't rotate the logo
- Don't change the logo colors
- Don't add effects (shadows, gradients, etc.)

## Color Palette

### Primary Colors

**Primary Blue**: `#1976d2`

- Used for primary actions, links, and interactive elements
- Theme color for PWA and browser chrome
- Accent color for headers

**Background**: `#1a1a1a`

- Dark theme primary background
- Used in PWA manifest

**Surface**: Defined in theme tokens

- Card and surface backgrounds
- Drawer backgrounds

**Text**: Defined in theme tokens

- Primary text color
- Muted text color for secondary information

### Usage Guidelines

**Buttons & CTAs**:

- Primary buttons: Primary blue background
- Secondary buttons: Outlined with primary blue
- Disabled: Reduced opacity

**Links**:

- Default: Primary blue
- Hover: Slightly darker shade
- Visited: Same as default (trading context)

**Status Colors**:

- Success/Profit: Green
- Error/Loss: Red
- Warning: Orange/Yellow
- Info: Blue

## Favicons

### Overview

The JAX Trading Assistant uses a complete set of favicons to ensure proper branding across all devices and platforms.

### Favicon Files

All favicon files are located in [`frontend/public/`](../frontend/public/):

| File | Size | Purpose |
| ---- | ---- | ------- |
| `favicon.ico` | 32x32 | Legacy browser support |
| `favicon-16x16.png` | 16x16 | Browser tab icon (small) |
| `favicon-32x32.png` | 32x32 | Browser tab icon (standard) |
| `favicon-192x192.png` | 192x192 | Android Chrome home screen |
| `favicon-512x512.png` | 512x512 | Android Chrome splash screen |
| `apple-touch-icon.png` | 180x180 | iOS home screen icon |

### Generating Favicons

#### Automated Generation

Use the provided PowerShell script to generate all favicon files from the SVG logo:

```powershell
.\scripts\generate-favicons.ps1
```

**Prerequisites**:

- Node.js installed
- Script will automatically install required dependencies

**What it does**:

1. Reads the source SVG logo
2. Converts to PNG at various sizes
3. Generates ICO file for legacy support
4. Places all files in `frontend/public/`

#### Manual Generation

If you prefer manual generation or the script fails, use one of these methods:

##### Option 1: Online Tool (Recommended for non-developers)

1. Visit [RealFaviconGenerator.net](https://realfavicongenerator.net/)
2. Upload `frontend/src/images/jax_ai_trader.svg`
3. Customize settings (use defaults)
4. Download the generated package
5. Extract files to `frontend/public/`

##### Option 2: ImageMagick (Command Line)

```bash
# Install ImageMagick first
# Then run these commands:

convert -background none frontend/src/images/jax_ai_trader.svg -resize 16x16 frontend/public/favicon-16x16.png
convert -background none frontend/src/images/jax_ai_trader.svg -resize 32x32 frontend/public/favicon-32x32.png
convert -background none frontend/src/images/jax_ai_trader.svg -resize 192x192 frontend/public/favicon-192x192.png
convert -background none frontend/src/images/jax_ai_trader.svg -resize 512x512 frontend/public/favicon-512x512.png
convert -background none frontend/src/images/jax_ai_trader.svg -resize 180x180 frontend/public/apple-touch-icon.png

# For ICO (multi-resolution):
convert frontend/public/favicon-16x16.png frontend/public/favicon-32x32.png frontend/public/favicon.ico
```

##### Option 3: Inkscape (GUI)

1. Open `frontend/src/images/jax_ai_trader.svg` in Inkscape
2. File → Export PNG Image
3. Set width and height to desired size
4. Export for each required size
5. Save files to `frontend/public/`

### Frontend Integration

#### HTML Head

The favicons are referenced in [`frontend/index.html`](../frontend/index.html):

```html
<!-- Favicons -->
<link rel="icon" type="image/x-icon" href="/favicon.ico" />
<link rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png" />
<link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png" />
<link rel="icon" type="image/png" sizes="192x192" href="/favicon-192x192.png" />
<link rel="icon" type="image/png" sizes="512x512" href="/favicon-512x512.png" />
<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png" />
```

#### PWA Manifest

The [`frontend/public/manifest.json`](../frontend/public/manifest.json) references the larger icons for Progressive Web App support:

```json
{
  "icons": [
    {
      "src": "/favicon-192x192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/favicon-512x512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any maskable"
    }
  ]
}
```

## Logo in Application

### Header Component

The logo is displayed in the application header ([`frontend/src/components/layout/AppShell.tsx`](../frontend/src/components/layout/AppShell.tsx)):

```tsx
import JaxLogo from '../../images/jax_ai_trader.svg';

// In the header:
<img 
  src={JaxLogo} 
  alt="Jax Trading Assistant Logo" 
  style={{ height: '32px', width: 'auto' }}
/>
```

**Best Practices**:

- Always include descriptive `alt` text for accessibility
- Use `height` with `width: auto` to maintain aspect ratio
- Make the logo clickable (link to home page)
- Reduce size on mobile devices

### Sidebar/Drawer

The logo also appears in the navigation drawer for branding consistency:

```tsx
<img 
  src={JaxLogo} 
  alt="Jax Logo" 
  style={{ height: '24px', width: 'auto' }}
/>
```

## Typography

### Font Family

The application uses system fonts for optimal performance and native feel:

- **Sans-serif stack**: System default fonts
- **Monospace**: For code, numbers, and technical data

### Font Weights

- **Regular**: 400 - Body text
- **Medium**: 500 - Subheadings
- **Semibold**: 600 - Section headers
- **Bold**: 700 - Primary headings

### Usage

Refer to [`frontend/src/styles/tokens.ts`](../frontend/src/styles/tokens.ts) for typography tokens:

```typescript
typography: {
  weight: {
    regular: 400,
    medium: 500,
    semibold: 600,
    bold: 700,
  }
}
```

## Updating Branding Assets

### When to Update

Consider updating branding assets when:

- Logo design changes
- New brand guidelines are established
- New platforms or devices need support
- Accessibility requirements change

### Update Process

1. **Update Source Logo**: Replace `frontend/src/images/jax_ai_trader.svg`

2. **Regenerate Favicons**: Run `.\scripts\generate-favicons.ps1`

3. **Update Components**: If logo usage changes, update:
   - `frontend/src/components/layout/AppShell.tsx`
   - Any other components using the logo

4. **Update Documentation**: Update this file with any new guidelines

5. **Test Across Devices**:
   - Desktop browsers (Chrome, Firefox, Safari, Edge)
   - Mobile browsers (iOS Safari, Android Chrome)
   - PWA installation
   - Different screen sizes

6. **Commit Changes**:

   ```bash
   git add frontend/src/images/jax_ai_trader.svg
   git add frontend/public/favicon*.png
   git add frontend/public/apple-touch-icon.png
   git add frontend/public/favicon.ico
   git commit -m "Update branding assets"
   ```

## Accessibility

### Color Contrast

- Ensure all text meets WCAG AA standards (4.5:1 for normal text)
- Test with accessibility tools (Chrome DevTools, Lighthouse)
- Provide sufficient contrast between logo and background

### Alternative Text

- Always include descriptive `alt` text for the logo
- Keep it concise but informative
- Example: "JAX Trading Assistant Logo"

### Responsive Design

- Logo scales appropriately on all screen sizes
- Touch targets are at least 44x44px on mobile
- Logo remains visible and recognizable when scaled

## File Structure

```text
jax-trading-assistant/
├── frontend/
│   ├── public/
│   │   ├── favicon.ico
│   │   ├── favicon-16x16.png
│   │   ├── favicon-32x32.png
│   │   ├── favicon-192x192.png
│   │   ├── favicon-512x512.png
│   │   ├── apple-touch-icon.png
│   │   └── manifest.json
│   ├── src/
│   │   ├── images/
│   │   │   └── jax_ai_trader.svg
│   │   └── components/
│   │       └── layout/
│   │           └── AppShell.tsx
│   └── index.html
├── scripts/
│   └── generate-favicons.ps1
└── Docs/
    └── BRANDING.md (this file)
```

## Resources

### Tools

- [RealFaviconGenerator](https://realfavicongenerator.net/) - Favicon generation
- [Favicon.io](https://favicon.io/) - Simple favicon generator
- [ImageMagick](https://imagemagick.org/) - Command-line image processing
- [Inkscape](https://inkscape.org/) - SVG editor

### Guidelines

- [Web.dev - Define icons](https://web.dev/add-manifest/#icons)
- [MDN - Favicon](https://developer.mozilla.org/en-US/docs/Learn/HTML/Introduction_to_HTML/The_head_metadata_in_HTML#adding_custom_icons_to_your_site)
- [Apple Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines/app-icons)

### Testing

- [Lighthouse](https://developers.google.com/web/tools/lighthouse) - PWA and performance
- [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/) - Color contrast
- [Favicon Checker](https://realfavicongenerator.net/favicon_checker) - Verify favicon implementation

## Changelog

### 2026-02-04

- Initial branding documentation
- Created favicon generation script
- Documented logo usage in AppShell component
- Established color palette guidelines
- Added favicon files to frontend/public/

---

For questions or suggestions about branding, please contact the development team or create an issue in the repository.
