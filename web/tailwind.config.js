/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js}'],
  theme: {
    extend: {
      colors: {
        bg: { DEFAULT: '#0f1117', card: '#161b22', hover: '#1c2128', input: '#0d1117' },
        border: { DEFAULT: '#30363d', light: '#21262d' },
        text: { DEFAULT: '#e1e4e8', muted: '#8b949e', dim: '#484f58' },
        accent: { blue: '#58a6ff', green: '#3fb950', yellow: '#d29922', red: '#f85149', purple: '#a371f7' },
        btn: { DEFAULT: '#21262d', hover: '#30363d', green: '#238636', greenHover: '#2ea043', red: '#da3633', redHover: '#f85149' }
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif']
      }
    }
  },
  plugins: []
}
