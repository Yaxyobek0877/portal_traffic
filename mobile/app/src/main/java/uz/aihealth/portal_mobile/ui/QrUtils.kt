package uz.aihealth.portal_mobile.ui

import android.graphics.Bitmap
import android.graphics.Color
import androidx.core.graphics.createBitmap
import com.google.zxing.BarcodeFormat
import com.google.zxing.EncodeHintType
import com.google.zxing.MultiFormatWriter

/** The on-the-wire invite format used by the desktop client. Mirroring it
 * lets the desktop's QR scan our QR and vice versa. See
 * client/frontend/src/components/PortalHeader.tsx (`inviteText`). */
fun inviteText(portalId: String, code: String): String =
    if (code.isNotEmpty()) "Portal\nID:   $portalId\nCode: $code"
    else "Portal\nID:   $portalId"

/**
 * Try to extract `(portalId, code)` from arbitrary text — handles:
 *   - the desktop's three-line `Portal\nID: …\nCode: …` format
 *   - simple `id-code`, `id code`, or `id · code` separator forms
 *
 * Returns null if we can't find two distinct 6-digit groups.
 */
fun parseInvite(raw: String): Pair<String, String>? {
    if (raw.isBlank()) return null
    val idMatch = Regex("""(?i)\bID\s*[:\-]?\s*(\d{6})""").find(raw)
    val codeMatch = Regex("""(?i)\b(?:Code|Kod)\s*[:\-]?\s*(\d{6})""").find(raw)
    if (idMatch != null && codeMatch != null) {
        return idMatch.groupValues[1] to codeMatch.groupValues[1]
    }
    // Fallback: pick the first two distinct 6-digit groups.
    val all = Regex("""\d{6}""").findAll(raw).map { it.value }.toList()
    val distinct = all.distinct()
    if (distinct.size >= 2) return distinct[0] to distinct[1]
    return null
}

/** Render the given text as a QR-code Bitmap of [sizePx] × [sizePx]. */
fun generateQrBitmap(text: String, sizePx: Int = 512): Bitmap {
    val hints = mapOf(EncodeHintType.MARGIN to 1)
    val matrix = MultiFormatWriter().encode(text, BarcodeFormat.QR_CODE, sizePx, sizePx, hints)
    val bmp = createBitmap(matrix.width, matrix.height, Bitmap.Config.RGB_565)
    for (x in 0 until matrix.width) {
        for (y in 0 until matrix.height) {
            bmp.setPixel(x, y, if (matrix[x, y]) Color.BLACK else Color.WHITE)
        }
    }
    return bmp
}
