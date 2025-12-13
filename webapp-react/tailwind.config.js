/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#5A7C52',
        'primary-dark': '#4A6B42',
        cream: '#FFE9AD',
        'cream-light': '#FDFBF5',
        sand: '#DBD0C4',
        brown: '#C8A067',
      },
      fontFamily: {
        main: ['Cormorant Garamond', 'Georgia', 'Times New Roman', 'serif'],
        secondary: ['Playfair Display', 'Georgia', 'serif'],
      },
    },
  },
  plugins: [],
}

