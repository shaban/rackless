import { Config } from 'tailwindcss'

export default {
  content: [
    "./frontend/**/*.{html,js,vue,ts}",
    "./frontend/*.{html,js,vue,ts}",
  ],
  darkMode: 'media', // Use system preference
  theme: {
    extend: {
      colors: {
        // Audio interface colors
        'audio-primary': '#3b82f6',    // Blue for primary controls
        'audio-secondary': '#e5e7eb',   // Light gray for secondary elements
        'audio-accent': '#60a5fa',      // Lighter blue for accents
        'audio-dark': '#1e293b',        // Dark blue for backgrounds
        'audio-surface': '#334155',     // Medium blue-gray for surfaces
        
        // Custom grays for dark interfaces
        'dark-surface': '#2a2a2a',
        'dark-border': '#404040',
        'dark-text': '#e0e0e0',
      },
      spacing: {
        // Common audio control sizes
        '32': '128px',  // 32x32 control size
        '64': '256px',  // Larger slider size
      },
      animation: {
        'control-glow': 'glow 2s ease-in-out infinite alternate',
      },
      keyframes: {
        glow: {
          '0%': { boxShadow: '0 0 5px rgba(59, 130, 246, 0.3)' },
          '100%': { boxShadow: '0 0 20px rgba(59, 130, 246, 0.6)' },
        }
      },
      gridTemplateColumns: {
        // Dynamic grid columns for layouts
        'audio-1': 'repeat(1, minmax(0, 1fr))',
        'audio-2': 'repeat(2, minmax(0, 1fr))',
        'audio-3': 'repeat(3, minmax(0, 1fr))',
        'audio-4': 'repeat(4, minmax(0, 1fr))',
        'audio-5': 'repeat(5, minmax(0, 1fr))',
        'audio-6': 'repeat(6, minmax(0, 1fr))',
        'audio-8': 'repeat(8, minmax(0, 1fr))',
        'audio-12': 'repeat(12, minmax(0, 1fr))',
      },
      gridTemplateRows: {
        // Dynamic grid rows for layouts  
        'audio-1': 'repeat(1, minmax(300px, auto))',
        'audio-2': 'repeat(2, minmax(300px, auto))',
        'audio-3': 'repeat(3, minmax(300px, auto))',
        'audio-4': 'repeat(4, minmax(300px, auto))',
        'audio-5': 'repeat(5, minmax(300px, auto))',
        'audio-6': 'repeat(6, minmax(300px, auto))',
      },
    },
  },
  plugins: [
    // Add any Tailwind plugins here if needed in the future
  ],
} satisfies Config
