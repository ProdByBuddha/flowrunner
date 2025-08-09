/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx}",
    "./components/**/*.{js,ts,jsx,tsx}"
  ],
  theme: {
    extend: {
      colors: {
        neon: {
          pink: '#ff2d95',
          cyan: '#00eaff',
          purple: '#9b5cf6',
          lime: '#b8ff00'
        },
        bg: {
          DEFAULT: '#0a0b0f',
          subtle: '#0e1016',
          panel: '#111321'
        }
      },
      boxShadow: {
        glow: '0 0 24px rgba(0, 234, 255, 0.25)',
        hard: '0 0 0 1px rgba(255,255,255,0.08)'
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'Noto Sans', 'Apple Color Emoji', 'Segoe UI Emoji']
      },
      backgroundImage: {
        grid: 'radial-gradient(circle at 1px 1px, rgba(255,255,255,0.06) 1px, transparent 0)'
      }
    },
  },
  plugins: [],
}
