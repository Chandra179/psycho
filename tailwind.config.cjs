// Colors read CSS variables (defined in web/app.css) so one stylesheet serves
// both the light and dark themes. `stone` is the themed neutral scale, `brand`
// the accent, and `white` the raised panel color.
const v = (name) => `rgb(var(--${name}) / <alpha-value>)`;
const scale = (prefix, shades) => Object.fromEntries(shades.map((s) => [s, v(`${prefix}-${s}`)]));

module.exports = {
  content: ['./templates/*.html', './modules/report/*.go', './site/*.html', './site/*.js'],
  theme: {
    extend: {
      colors: {
        white: v('panel'),
        stone: scale('stone', [50, 100, 200, 300, 400, 500, 600, 700, 800, 900]),
        brand: scale('brand', [50, 100, 200, 500, 600, 700, 800]),
        amber: scale('amber', [50, 200, 300, 800, 900]),
        rose: scale('rose', [50, 200, 300, 500, 800]),
      },
    },
  },
  plugins: [],
};
