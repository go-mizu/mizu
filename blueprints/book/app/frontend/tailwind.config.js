/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        serif: ['Merriweather', 'Georgia', 'serif'],
        sans: ['Lato', 'system-ui', 'sans-serif'],
      },
      colors: {
        gr: {
          brown: '#382110',
          tan: '#F4F1EA',
          cream: '#FBF9F4',
          teal: '#00635D',
          orange: '#E87400',
          green: '#409D69',
          border: '#D8D8D8',
          text: '#333333',
          light: '#999999',
          hover: '#F0EDDF',
        },
      },
    },
  },
  plugins: [],
}
