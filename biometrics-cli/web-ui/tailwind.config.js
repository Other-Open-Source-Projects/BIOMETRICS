/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./index.html", "./static/js/**/*.js"],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        dark: '#0f172a',
        darker: '#0b1120',
        brand: {
          500: '#7D56F4',
          600: '#6b46c1',
          700: '#553c9a',
        },
        neon: '#04B575',
        alert: '#F56565'
      }
    }
  },
  plugins: [],
}
