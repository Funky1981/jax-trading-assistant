# Styling & Theming

## Centralized Styling Strategy
- Use design tokens as the single source of truth for spacing, color, typography, elevation, and motion.
- Avoid ad‑hoc styles in individual components.
- Themes should be applied at the app root and cascaded through tokens.

## Design Tokens
- **Color:** background, surface, accent, critical, warning, success.
- **Typography:** font family, scale, weight, line height.
- **Spacing:** 4‑pt or 8‑pt grid.
- **Elevation:** small set of shadow levels.

## Themes
- **Default:** dark trading theme optimized for high contrast.
- **Alternate:** light theme for daylight environments.
- **Accessibility:** ensure contrast ratios meet WCAG AA/AAA for dense data.

## Chart Palette
- Define consistent color mapping for:
  - Up/down movement
  - Volume intensity
  - Risk severity

## Styling Best Practices
- Keep CSS tightly scoped to components.
- Prefer utility tokens over hardcoded values.
- No component should define its own color palette.

