/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#2E7FF0',
          50: '#E9F1FF',
          100: '#D7E6FF',
          200: '#B3D0FF',
          300: '#8AB8FF',
          400: '#5A97F7',
          500: '#2E7FF0',
          600: '#1D63D6',
          700: '#184CAF',
          800: '#143A82',
          900: '#0E2457',
        },
        success: '#20C933',
        danger: '#F04438',
        warning: '#FCAE40',
        purple: '#8B46FF',
      },
      fontFamily: {
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', 'sans-serif'],
      },
    },
  },
  plugins: [],
};
