// MyCal's own app mark. Same geometry as web/static/favicon.svg — identical
// rects, identical text placement — but every fill and stroke is `currentColor`
// instead of the favicon's blue-on-white, so the mark inverts to white on a
// --primary square the way MyMail draws its sidebar badge.
//
// That inversion is the ONLY intended difference: the favicon fills the body
// rect white, this one leaves it open so the badge colour shows through. Any
// other change here belongs in favicon.svg too, and vice versa.
//
// It lives here rather than in <Icon> because it is not a Lucide icon and is
// not in the vendored bundle — do not try to route it through gen-lucide.mjs.
//
// Size comes from CSS (`.brand-logo svg`), so the badge's box and its contents
// are sized in one place.

export function Logo() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" fill="none" aria-hidden="true">
      <rect x="2" y="5" width="28" height="25" rx="3" stroke="currentColor" stroke-width="2" />
      <rect x="2" y="5" width="28" height="8" rx="3" fill="currentColor" />
      <rect x="8" y="2" width="3" height="6" rx="1" fill="currentColor" />
      <rect x="21" y="2" width="3" height="6" rx="1" fill="currentColor" />
      <text
        x="16"
        y="25"
        text-anchor="middle"
        font-family="Arial,sans-serif"
        font-size="12"
        font-weight="bold"
        fill="currentColor"
      >
        8
      </text>
    </svg>
  );
}
