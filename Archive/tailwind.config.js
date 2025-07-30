/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./frontend/**/*.{html,js,vue}",
    "./frontend/*.{html,js,vue}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#3b82f6',
        secondary: '#e5e7eb',
      }
    },
  },
  plugins: [],
}
