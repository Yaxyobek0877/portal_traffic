package uz.aihealth.portal_mobile.ui

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import kotlin.math.min

// Compose-native port of client/frontend/src/components/Logo.tsx — five
// concentric rings with the same dash patterns and rotation directions
// the desktop SVG uses, plus a soft violet glow + white centre dot. The
// rings are infiniteTransition-animated so the welcome screen has the
// same "wormhole alive" feel as the Wails app.
//
// We deliberately use a single PathEffect-based dashed stroke per ring
// rather than rendering individual arc segments — Compose's Canvas is
// fast enough that the visual is identical with much less code, and
// PathEffect handles antialiasing better at small sizes than manual
// arc strokes.
@Composable
fun Logo(size: Dp = 170.dp, animated: Boolean = true) {
    val transition = rememberInfiniteTransition(label = "logo")

    // Per-ring spin speed and direction mirror the SVG: outer rings
    // rotate slowly, inner rings fast; alternate clockwise / CCW so
    // they don't visually merge into a single wheel.
    val ring0 = transition.animatedAngle(durationMs = 14_000, reverse = false, enabled = animated)
    val ring1 = transition.animatedAngle(durationMs = 9_000, reverse = true, enabled = animated)
    val ring2 = transition.animatedAngle(durationMs = 6_000, reverse = false, enabled = animated)
    val ring3 = transition.animatedAngle(durationMs = 4_000, reverse = true, enabled = animated)
    val ring4 = transition.animatedAngle(durationMs = 2_500, reverse = false, enabled = animated)

    // Centre node "pulse" — subtle scale on the white dot so the eye
    // settles there. 2.4s matches the SVG's nodePulse keyframes.
    val pulse by transition.animateFloat(
        initialValue = 0.92f,
        targetValue = 1.08f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 2_400, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "pulse",
    )

    val brush = Brush.linearGradient(
        colors = listOf(
            Color(0xFF8B5CF6),  // violet-500
            Color(0xFF6366F1),  // indigo-500
            Color(0xFF22D3EE),  // cyan-400
        ),
    )

    Canvas(modifier = Modifier.size(size)) {
        val w = this.size.minDimension
        val center = Offset(this.size.width / 2f, this.size.height / 2f)
        // Logo viewBox is 200; scale our pixel canvas accordingly.
        val s = w / 200f

        drawGlow(center, radiusPx = 28f * s, color = Color(0xFF8B5CF6))

        drawRing(center, radius = 92f * s, stroke = 1.2f * s, brush = brush, alpha = 0.55f, dash = floatArrayOf(3f * s, 9f * s), rotation = ring0)
        drawRing(center, radius = 78f * s, stroke = 1.4f * s, brush = brush, alpha = 0.70f, dash = floatArrayOf(6f * s, 4f * s), rotation = ring1)
        drawRing(center, radius = 62f * s, stroke = 1.7f * s, brush = brush, alpha = 0.82f, dash = floatArrayOf(2f * s, 6f * s), rotation = ring2)
        drawRing(center, radius = 46f * s, stroke = 2.1f * s, brush = brush, alpha = 1.0f, dash = floatArrayOf(14f * s, 6f * s), rotation = ring3)
        drawRing(center, radius = 30f * s, stroke = 1.6f * s, brush = brush, alpha = 0.65f, dash = null, rotation = ring4)

        // Centre dot. Drawn last so the glow doesn't muddy it.
        val dotR = 6f * s * pulse
        drawCircle(color = Color.White, radius = dotR, center = center, alpha = 0.95f)
    }
}

private fun DrawScope.drawGlow(center: Offset, radiusPx: Float, color: Color) {
    // Cheap radial-gradient simulation: stack three translucent fills
    // with decreasing alpha. Compose has Brush.radialGradient but it
    // burns more frame time on cheap GPUs than concentric solid fills.
    drawCircle(color = color.copy(alpha = 0.18f), radius = radiusPx * 1.6f, center = center)
    drawCircle(color = color.copy(alpha = 0.32f), radius = radiusPx * 1.0f, center = center)
    drawCircle(color = color.copy(alpha = 0.55f), radius = radiusPx * 0.55f, center = center)
}

private fun DrawScope.drawRing(
    center: Offset,
    radius: Float,
    stroke: Float,
    brush: Brush,
    alpha: Float,
    dash: FloatArray?,
    rotation: Float,
) {
    if (radius <= 0f || stroke <= 0f) return
    val effect = dash?.let { PathEffect.dashPathEffect(it, 0f) }
    rotate(rotation, pivot = center) {
        val path = Path().apply {
            addOval(
                Rect(
                    left = center.x - radius,
                    top = center.y - radius,
                    right = center.x + radius,
                    bottom = center.y + radius,
                ),
            )
        }
        drawPath(
            path = path,
            brush = brush,
            alpha = alpha,
            style = Stroke(width = stroke, pathEffect = effect),
        )
    }
}

@Composable
private fun androidx.compose.animation.core.InfiniteTransition.animatedAngle(
    durationMs: Int,
    reverse: Boolean,
    enabled: Boolean,
): Float {
    if (!enabled) return 0f
    val v by animateFloat(
        initialValue = if (reverse) 360f else 0f,
        targetValue = if (reverse) 0f else 360f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = durationMs, easing = LinearEasing),
            repeatMode = RepeatMode.Restart,
        ),
        label = "spin-$durationMs",
    )
    return v
}
