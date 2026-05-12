export const TYPOGRAPHY = {
  fontFamily:
    '"Inter", "Noto Sans CJK SC", "Microsoft YaHei", "PingFang SC", sans-serif',
  color: '#333333',
  base: { fontSize: '14px', lineHeight: 1.5, fontWeight: 400 },
  h1: { fontSize: '24px', lineHeight: 1.3, fontWeight: 600 },
  h2: { fontSize: '20px', lineHeight: 1.3, fontWeight: 600 },
  h3: { fontSize: '16px', lineHeight: 1.4, fontWeight: 500 },
  p: { fontSize: '14px', lineHeight: 1.5, fontWeight: 400, minHeight: '1.5em' },
  ul: { paddingLeft: '24px', listStyleType: 'disc' },
  ol: { paddingLeft: '24px', listStyleType: 'decimal' },
  li: { marginBottom: '4px' },
} as const
