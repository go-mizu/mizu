/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#2D7FF9',
          50: '#E8F1FE',
          100: '#CFDFFF',
          200: '#A0BFFF',
          300: '#719FFF',
          400: '#4E8BF9',
          500: '#2D7FF9',
          600: '#0A64E0',
          700: '#084BB0',
          800: '#063580',
          900: '#041F50',
        },
        success: '#20C933',
        danger: '#F82B60',
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
