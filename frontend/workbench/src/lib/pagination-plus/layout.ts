/** A4 page layout constants at 96dpi (single source of truth) */

export const A4_LAYOUT = {
  /** 210mm @ 96dpi */
  pageWidth: 794,
  /** 297mm @ 96dpi */
  pageHeight: 1123,
  /** ≈ 11mm top */
  marginTop: 42,
  /** 18mm bottom ≈ 68px */
  marginBottom: 68,
  /** 20mm left ≈ 76px */
  marginLeft: 76,
  /** 20mm right ≈ 76px */
  marginRight: 76,
} as const

/** Total canvas width: page + left margin + right margin */
export const CANVAS_TOTAL_WIDTH_PX = A4_LAYOUT.pageWidth + A4_LAYOUT.marginLeft + A4_LAYOUT.marginRight
