import type { Config } from 'tailwindcss';

const config: Config = {
  darkMode: ['class'],
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      // ─── PT Manager Design System Tokens ──────────────────────
      // Dark-first palette inspired by Linear / Vercel / Raycast
      colors: {
        // ── Backgrounds — CSS 变量驱动，支持 light / dark ──
        bg: {
          base:     'hsl(var(--pt-bg-base))',
          card:     'hsl(var(--pt-bg-card))',
          elevated: 'hsl(var(--pt-bg-elevated))',
          overlay:  'hsl(var(--pt-bg-overlay))',
        },
        // ── Borders ──
        border: {
          DEFAULT: 'hsl(var(--pt-border))',
          subtle:  'hsl(var(--pt-border-subtle))',
          focus:   '#6366F1',
        },
        // ── Text ──
        fg: {
          DEFAULT: 'hsl(var(--pt-fg))',
          muted:   'hsl(var(--pt-fg-muted))',
          subtle:  'hsl(var(--pt-fg-subtle))',
        },
        // ── Accent — Indigo/Violet ──
        accent: {
          DEFAULT: '#6366F1',   // Indigo-500
          hover: '#4F46E5',     // Indigo-600
          muted: '#312E81',     // dark tint
          violet: '#8B5CF6',    // Violet-500
        },
        // ── Semantic ──
        success: { DEFAULT: '#22C55E', muted: '#14532D' },
        warning: { DEFAULT: '#F59E0B', muted: '#451A03' },
        danger:  { DEFAULT: '#EF4444', muted: '#450A0A' },
        info:    { DEFAULT: '#38BDF8', muted: '#0C4A6E' },
        // ── shadcn/ui CSS variable bindings ──
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
      },
      // ─── Border Radius ────────────────────────────────────────
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      // ─── Font Family ──────────────────────────────────────────
      fontFamily: {
        sans: ['var(--font-geist-sans)', 'system-ui', 'sans-serif'],
        mono: ['var(--font-geist-mono)', 'monospace'],
      },
      // ─── Keyframes ────────────────────────────────────────────
      keyframes: {
        'fade-in': { '0%': { opacity: '0' }, '100%': { opacity: '1' } },
        'slide-up': {
          '0%': { transform: 'translateY(8px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        pulse: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.5' },
        },
        // 移动端底部抽屉 / bottom sheet
        'sheet-up': {
          '0%': { transform: 'translateY(100%)' },
          '100%': { transform: 'translateY(0)' },
        },
        'sheet-down': {
          '0%': { transform: 'translateY(0)' },
          '100%': { transform: 'translateY(100%)' },
        },
        'fade-out': { '0%': { opacity: '1' }, '100%': { opacity: '0' } },
      },
      animation: {
        'fade-in': 'fade-in 0.2s ease-out',
        'fade-out': 'fade-out 0.15s ease-in',
        'slide-up': 'slide-up 0.3s ease-out',
        'sheet-up': 'sheet-up 0.28s cubic-bezier(0.32, 0.72, 0, 1)',
        'sheet-down': 'sheet-down 0.2s ease-in',
      },
    },
  },
  plugins: [],
};

export default config;
