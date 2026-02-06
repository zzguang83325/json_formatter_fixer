/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#3498db',
        secondary: '#e67e22',
        success: '#2ecc71',
        error: '#e74c3c',
      }
    },
  },
  plugins: [],
}
