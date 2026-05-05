/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#faf6f2',
          100: '#f2e8e0',
          200: '#e5d2c2',
          300: '#d4b99a',
          400: '#c4956a',
          500: '#b3804d',
          600: '#9c6b3a',
          700: '#7d5530',
          800: '#5e4025',
          900: '#3f2b1a',
          950: '#2a1c10',
        },
        background: '#faf8f5',
        foreground: '#1a1815',
        card: {
          DEFAULT: '#ffffff',
          foreground: '#1a1815',
        },
        muted: {
          DEFAULT: '#f5f1ed',
          foreground: '#8c8279',
        },
        secondary: {
          DEFAULT: '#e8d5c4',
          foreground: '#5c4a3a',
        },
        accent: {
          DEFAULT: '#d4a574',
          foreground: '#4a3020',
        },
        destructive: {
          DEFAULT: '#d64545',
          foreground: '#ffffff',
        },
        border: '#e4ddd5',
        input: '#e4ddd5',
        ring: '#c4956a',
      },
      fontFamily: {
        serif: ['Playfair Display', 'Noto Serif SC', 'STSong', 'serif'],
        sans: ['Inter', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      keyframes: {
        'fade-in-up': {
          '0%': { opacity: '0', transform: 'translateY(20px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'scale-in': {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' },
        },
        'float': {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-10px)' },
        },
      },
      animation: {
        'fade-in-up': 'fade-in-up 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards',
        'fade-in': 'fade-in 0.5s ease-out forwards',
        'scale-in': 'scale-in 0.5s ease-out forwards',
        'float': 'float 6s ease-in-out infinite',
        'float-delayed': 'float 8s ease-in-out infinite',
      },
    },
  },
}
