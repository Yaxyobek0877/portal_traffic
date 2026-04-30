package uz.aihealth.portal_mobile.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class QrUtilsTest {

    @Test fun `parses desktop multi-line format`() {
        val raw = "Portal\nID:   428591\nCode: 739204"
        val (id, code) = parseInvite(raw)!!
        assertEquals("428591", id)
        assertEquals("739204", code)
    }

    @Test fun `parses Uzbek 'Kod' alias`() {
        val raw = "Portal\nID: 428591\nKod: 739204"
        val (id, code) = parseInvite(raw)!!
        assertEquals("428591", id)
        assertEquals("739204", code)
    }

    @Test fun `case-insensitive labels`() {
        val raw = "id: 428591\ncode: 739204"
        val (id, code) = parseInvite(raw)!!
        assertEquals("428591", id)
        assertEquals("739204", code)
    }

    @Test fun `falls back to two distinct 6-digit groups`() {
        // No labels — just digits separated by something.
        assertEquals("428591" to "739204", parseInvite("428591 · 739204"))
        assertEquals("428591" to "739204", parseInvite("428591-739204"))
        assertEquals("428591" to "739204", parseInvite("Portal 428591 / 739204"))
    }

    @Test fun `rejects garbage`() {
        assertNull(parseInvite(""))
        assertNull(parseInvite("hello world"))
        assertNull(parseInvite("12345"))           // only 5 digits
        assertNull(parseInvite("428591"))          // only one 6-digit group
        assertNull(parseInvite("428591 428591"))   // two groups but identical, not distinct
    }

    @Test fun `inviteText round-trips through parseInvite`() {
        val text = inviteText("428591", "739204")
        val parsed = parseInvite(text)!!
        assertEquals("428591", parsed.first)
        assertEquals("739204", parsed.second)
    }

    @Test fun `inviteText without code only contains ID`() {
        val text = inviteText("428591", "")
        // Should be parseable as ID-only (returns null because we need two
        // groups), but contain the ID literally.
        assert(text.contains("428591"))
        assertNull(parseInvite(text))
    }
}
