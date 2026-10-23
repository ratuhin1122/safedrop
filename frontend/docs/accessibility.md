# Accessibility checks

Run npm run audit:a11y. It builds with VITE_API_URL=/v1 for the fixture server.
Set CHROME_PATH to your Chrome or Edge executable if discovery fails.

The audit starts an isolated local server with a synthetic signed-in account,
usage above 80%, and ready, scanning, and rejected files. It audits mobile
emulation on both routes in both themes, fails below 95, saves HTML reports
under .lighthouse/, and stops its server and browser when complete.
These fixture results are not a production deployment audit.

Measured with Lighthouse 13.5.0 on 2026-10-23:

| Route | Light | Dark |
| --- | --- | --- |
| / | 100 | 100 |
| /files | 100 | 100 |

Alt + Shift + U opens the upload picker. The shortcut ignores text editors,
open dialogs, repeat/composition events, and quota-disabled selection.
Browse files remains available for keyboard, touch, and assistive technology.

Upload announcements use polite ten-percent progress updates; errors use
assertive alerts. Images use native lazy loading and async decoding; thumbnail
requests wait until near the viewport. Reduced-motion preferences disable
animations and transitions. Visible focus outlines apply across the app.

Manual checks still matter: tab and reverse-tab through controls, activate
the skip link, navigate menus with arrows and Escape, use file and confirmation
dialogs, verify live announcements with a screen reader, and test zoom at 200%.
The automated score does not certify every interactive state.
