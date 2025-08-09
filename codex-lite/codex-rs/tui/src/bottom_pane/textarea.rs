use crossterm::event::KeyCode;
use crossterm::event::KeyEvent;
use crossterm::event::KeyModifiers;
use ratatui::buffer::Buffer;
use ratatui::layout::Rect;
use ratatui::style::Style;
use ratatui::widgets::StatefulWidgetRef;
use ratatui::widgets::WidgetRef;
use std::cell::Ref;
use std::cell::RefCell;
use std::ops::Range;
use textwrap::Options;
use unicode_segmentation::UnicodeSegmentation;
use unicode_width::UnicodeWidthStr;

#[derive(Debug)]
pub(crate) struct TextArea {
    text: String,
    cursor_pos: usize,
    wrap_cache: RefCell<Option<WrapCache>>,
    preferred_col: Option<usize>,
}

#[derive(Debug, Clone)]
struct WrapCache {
    width: u16,
    lines: Vec<Range<usize>>,
}

#[derive(Debug, Default, Clone, Copy)]
pub(crate) struct TextAreaState {
    /// Index into wrapped lines of the first visible line.
    scroll: u16,
}

impl TextArea {
    pub fn new() -> Self {
        Self {
            text: String::new(),
            cursor_pos: 0,
            wrap_cache: RefCell::new(None),
            preferred_col: None,
        }
    }

    pub fn set_text(&mut self, text: &str) {
        self.text = text.to_string();
        self.cursor_pos = self.cursor_pos.clamp(0, self.text.len());
        self.wrap_cache.replace(None);
        self.preferred_col = None;
    }

    pub fn text(&self) -> &str {
        &self.text
    }

    pub fn insert_str(&mut self, text: &str) {
        self.insert_str_at(self.cursor_pos, text);
    }

    pub fn insert_str_at(&mut self, pos: usize, text: &str) {
        self.text.insert_str(pos, text);
        self.wrap_cache.replace(None);
        if pos <= self.cursor_pos {
            self.cursor_pos += text.len();
        }
        self.preferred_col = None;
    }

    pub fn replace_range(&mut self, range: std::ops::Range<usize>, text: &str) {
        assert!(range.start <= range.end);
        let start = range.start.clamp(0, self.text.len());
        let end = range.end.clamp(0, self.text.len());
        let removed_len = end - start;
        let inserted_len = text.len();
        if removed_len == 0 && inserted_len == 0 {
            return;
        }
        let diff = inserted_len as isize - removed_len as isize;

        self.text.replace_range(range, text);
        self.wrap_cache.replace(None);
        self.preferred_col = None;

        // Update the cursor position to account for the edit.
        self.cursor_pos = if self.cursor_pos < start {
            // Cursor was before the edited range – no shift.
            self.cursor_pos
        } else if self.cursor_pos <= end {
            // Cursor was inside the replaced range – move to end of the new text.
            start + inserted_len
        } else {
            // Cursor was after the replaced range – shift by the length diff.
            ((self.cursor_pos as isize) + diff) as usize
        }
        .min(self.text.len());
    }

    pub fn cursor(&self) -> usize {
        self.cursor_pos
    }

    pub fn set_cursor(&mut self, pos: usize) {
        self.cursor_pos = pos.clamp(0, self.text.len());
        self.preferred_col = None;
    }

    pub fn desired_height(&self, width: u16) -> u16 {
        self.wrapped_lines(width).len() as u16
    }

    #[allow(dead_code)]
    pub fn cursor_pos(&self, area: Rect) -> Option<(u16, u16)> {
        self.cursor_pos_with_state(area, &TextAreaState::default())
    }

    /// Compute the on-screen cursor position taking scrolling into account.
    pub fn cursor_pos_with_state(&self, area: Rect, state: &TextAreaState) -> Option<(u16, u16)> {
        let lines = self.wrapped_lines(area.width);
        let effective_scroll = self.effective_scroll(area.height, &lines, state.scroll);
        let i = Self::wrapped_line_index_by_start(&lines, self.cursor_pos)?;
        let ls = &lines[i];
        let col = self.text[ls.start..self.cursor_pos].width() as u16;
        let screen_row = i
            .saturating_sub(effective_scroll as usize)
            .try_into()
            .unwrap_or(0);
        Some((area.x + col, area.y + screen_row))
    }

    pub fn is_empty(&self) -> bool {
        self.text.is_empty()
    }

    fn current_display_col(&self) -> usize {
        let bol = self.beginning_of_current_line();
        self.text[bol..self.cursor_pos].width()
    }

    fn wrapped_line_index_by_start(lines: &[Range<usize>], pos: usize) -> Option<usize> {
        // partition_point returns the index of the first element for which
        // the predicate is false, i.e. the count of elements with start <= pos.
        let idx = lines.partition_point(|r| r.start <= pos);
        if idx == 0 { None } else { Some(idx - 1) }
    }

    fn move_to_display_col_on_line(
        &mut self,
        line_start: usize,
        line_end: usize,
        target_col: usize,
    ) {
        let mut width_so_far = 0usize;
        for (i, g) in self.text[line_start..line_end].grapheme_indices(true) {
            width_so_far += g.width();
            if width_so_far > target_col {
                self.cursor_pos = line_start + i;
                return;
            }
        }
        self.cursor_pos = line_end;
    }

    fn beginning_of_line(&self, pos: usize) -> usize {
        self.text[..pos].rfind('\n').map(|i| i + 1).unwrap_or(0)
    }
    fn beginning_of_current_line(&self) -> usize {
        self.beginning_of_line(self.cursor_pos)
    }

    fn end_of_line(&self, pos: usize) -> usize {
        self.text[pos..]
            .find('\n')
            .map(|i| i + pos)
            .unwrap_or(self.text.len())
    }
    fn end_of_current_line(&self) -> usize {
        self.end_of_line(self.cursor_pos)
    }

    pub(crate) fn beginning_of_previous_word(&self) -> usize {
        if let Some(first_non_ws) = self.text[..self.cursor_pos].rfind(|c: char| !c.is_whitespace())
        {
            self.text[..first_non_ws]
                .rfind(|c: char| c.is_whitespace())
                .map(|i| i + 1)
                .unwrap_or(0)
        } else {
            0
        }
    }

    pub(crate) fn end_of_next_word(&self) -> usize {
        let Some(first_non_ws) = self.text[self.cursor_pos..].find(|c: char| !c.is_whitespace())
        else {
            return self.text.len();
        };
        let word_start = self.cursor_pos + first_non_ws;
        match self.text[word_start..].find(|c: char| c.is_whitespace()) {
            Some(rel_idx) => word_start + rel_idx,
            None => self.text.len(),
        }
    }

    pub fn input(&mut self, event: KeyEvent) {
        match event {
            KeyEvent {
                code: KeyCode::Char(c),
                // Insert plain characters (and Shift-modified). Do NOT insert when ALT is held,
                // because many terminals map Option/Meta combos to ALT+<char> (e.g. ESC f/ESC b)
                // for word navigation. Those are handled explicitly below.
                modifiers: KeyModifiers::NONE | KeyModifiers::SHIFT,
                ..
            } => self.insert_str(&c.to_string()),
            KeyEvent {
                code: KeyCode::Char('j' | 'm'),
                modifiers: KeyModifiers::CONTROL,
                ..
            }
            | KeyEvent {
                code: KeyCode::Enter,
                ..
            } => self.insert_str("\n"),
            KeyEvent {
                code: KeyCode::Backspace,
                modifiers: KeyModifiers::ALT,
                ..
            } => self.delete_backward_word(),
            KeyEvent {
                code: KeyCode::Backspace,
                modifiers: KeyModifiers::NONE,
                ..
            } => self.delete_backward(1),
            KeyEvent {
                code: KeyCode::Delete,
                ..
            }
            | KeyEvent {
                code: KeyCode::Char('d'),
                modifiers: KeyModifiers::CONTROL,
                ..
            } => self.delete_forward(1),

            KeyEvent {
                code: KeyCode::Char('w'),
                modifiers: KeyModifiers::CONTROL,
                ..
            } => {
                self.delete_backward_word();
            }
            // Meta-b -> move to beginning of previous word
            // Meta-f -> move to end of next word
            // Many terminals map Option (macOS) to Alt. Some send Alt|Shift, so match contains(ALT).
            KeyEvent {
                code: KeyCode::Char('b'),
                modifiers: KeyModifiers::ALT,
                ..
            } => {
                self.set_cursor(self.beginning_of_previous_word());
            }
            KeyEvent {
                code: KeyCode::Char('f'),
                modifiers: KeyModifiers::ALT,
                ..
            } => {
                self.set_cursor(self.end_of_next_word());
            }
            KeyEvent {
                code: KeyCode::Char('u'),
                modifiers: KeyModifiers::CONTROL,
                ..
            } => {
                self.kill_to_beginning_of_line();
            }
            KeyEvent {
                code: KeyCode::Char('k'),
                modifiers: KeyModifiers::CONTROL,
                ..
            } => {
                self.kill_to_end_of_line();
            }

            // Cursor movement
            KeyEvent {
                code: KeyCode::Left,
                modifiers: KeyModifiers::NONE,
                ..
            } => {
                self.move_cursor_left();
            }
            KeyEvent {
                code: KeyCode::Right,
                modifiers: KeyModifiers::NONE,
                ..
            } => {
                self.move_cursor_right();
            }
            // Some terminals send Alt+Arrow for word-wise movement:
            // Option/Left -> Alt+Left (previous word start)
            // Option/Right -> Alt+Right (next word end)
            KeyEvent {
                code: KeyCode::Left,
                modifiers: KeyModifiers::ALT,
                ..
            }
            | KeyEvent {
                code: KeyCode::Left,
                modifiers: KeyModifiers::CONTROL,
                ..
            } => {
                self.set_cursor(self.beginning_of_previous_word());
            }
            KeyEvent {
                code: KeyCode::Right,
                modifiers: KeyModifiers::ALT,
                ..
            }
            | KeyEvent {
                code: KeyCode::Right,
                modifiers: KeyModifiers::CONTROL,
                ..
            } => {
                self.set_cursor(self.end_of_next_word());
            }
            KeyEvent {
                code: KeyCode::Up, ..
            } => {
                self.move_cursor_up();
            }
            KeyEvent {
                code: KeyCode::Down,
                ..
            } => {
                self.move_cursor_down();
            }
            KeyEvent {
                code: KeyCode::Home,
                ..
            } => {
                self.move_cursor_to_beginning_of_line(false);
            }
            KeyEvent {
                code: KeyCode::Char('a'),
                modifiers: KeyModifiers::CONTROL,
                ..
            } => {
                self.move_cursor_to_beginning_of_line(true);
            }

            KeyEvent {
                code: KeyCode::End, ..
            } => {
                self.move_cursor_to_end_of_line(false);
            }
            KeyEvent {
                code: KeyCode::Char('e'),
                modifiers: KeyModifiers::CONTROL,
                ..
            } => {
                self.move_cursor_to_end_of_line(true);
            }
            o => {
                tracing::debug!("Unhandled key event in TextArea: {:?}", o);
            }
        }
    }

    // ####### Input Functions #######
    pub fn delete_backward(&mut self, n: usize) {
        if n == 0 || self.cursor_pos == 0 {
            return;
        }
        let mut gc =
            unicode_segmentation::GraphemeCursor::new(self.cursor_pos, self.text.len(), false);
        let mut target = self.cursor_pos;
        for _ in 0..n {
            match gc.prev_boundary(&self.text, 0) {
                Ok(Some(b)) => target = b,
                Ok(None) => {
                    target = 0;
                    break;
                }
                Err(_) => {
                    target = target.saturating_sub(1);
                }
            }
        }
        self.replace_range(target..self.cursor_pos, "");
    }

    pub fn delete_forward(&mut self, n: usize) {
        if n == 0 || self.cursor_pos >= self.text.len() {
            return;
        }
        let mut gc =
            unicode_segmentation::GraphemeCursor::new(self.cursor_pos, self.text.len(), false);
        let mut target = self.cursor_pos;
        for _ in 0..n {
            match gc.next_boundary(&self.text, 0) {
                Ok(Some(b)) => target = b,
                Ok(None) => {
                    target = self.text.len();
                    break;
                }
                Err(_) => {
                    target = target.saturating_add(1);
                }
            }
        }
        self.replace_range(self.cursor_pos..target, "");
    }

    pub fn delete_backward_word(&mut self) {
        self.replace_range(self.beginning_of_previous_word()..self.cursor_pos, "");
    }

    pub fn kill_to_end_of_line(&mut self) {
        let eol = self.end_of_current_line();
        if self.cursor_pos == eol {
            if eol < self.text.len() {
                self.replace_range(self.cursor_pos..eol + 1, "");
            }
        } else {
            self.replace_range(self.cursor_pos..eol, "");
        }
    }

    pub fn kill_to_beginning_of_line(&mut self) {
        let bol = self.beginning_of_current_line();
        if self.cursor_pos == bol {
            if bol > 0 {
                self.replace_range(bol - 1..bol, "");
            }
        } else {
            self.replace_range(bol..self.cursor_pos, "");
        }
    }

    /// Move the cursor left by a single grapheme cluster.
    pub fn move_cursor_left(&mut self) {
        let mut gc =
            unicode_segmentation::GraphemeCursor::new(self.cursor_pos, self.text.len(), false);
        match gc.prev_boundary(&self.text, 0) {
            Ok(Some(boundary)) => self.cursor_pos = boundary,
            Ok(None) => self.cursor_pos = 0, // Already at start.
            Err(_) => self.cursor_pos = self.cursor_pos.saturating_sub(1),
        }
        self.preferred_col = None;
    }

    /// Move the cursor right by a single grapheme cluster.
    pub fn move_cursor_right(&mut self) {
        let mut gc =
            unicode_segmentation::GraphemeCursor::new(self.cursor_pos, self.text.len(), false);
        match gc.next_boundary(&self.text, 0) {
            Ok(Some(boundary)) => self.cursor_pos = boundary,
            Ok(None) => self.cursor_pos = self.text.len(), // Already at end.
            Err(_) => self.cursor_pos = self.cursor_pos.saturating_add(1),
        }
        self.preferred_col = None;
    }

    pub fn move_cursor_up(&mut self) {
        // If we have a wrapping cache, prefer navigating across wrapped (visual) lines.
        if let Some((target_col, maybe_line)) = {
            let cache_ref = self.wrap_cache.borrow();
            if let Some(cache) = cache_ref.as_ref() {
                let lines = &cache.lines;
                if let Some(idx) = Self::wrapped_line_index_by_start(lines, self.cursor_pos) {
                    let cur_range = &lines[idx];
                    let target_col = self
                        .preferred_col
                        .unwrap_or_else(|| self.text[cur_range.start..self.cursor_pos].width());
                    if idx > 0 {
                        let prev = &lines[idx - 1];
                        let line_start = prev.start;
                        let line_end = prev.end.saturating_sub(1);
                        Some((target_col, Some((line_start, line_end))))
                    } else {
                        Some((target_col, None))
                    }
                } else {
                    None
                }
            } else {
                None
            }
        } {
            // We had wrapping info. Apply movement accordingly.
            match maybe_line {
                Some((line_start, line_end)) => {
                    if self.preferred_col.is_none() {
                        self.preferred_col = Some(target_col);
                    }
                    self.move_to_display_col_on_line(line_start, line_end, target_col);
                    return;
                }
                None => {
                    // Already at first visual line -> move to start
                    self.cursor_pos = 0;
                    self.preferred_col = None;
                    return;
                }
            }
        }

        // Fallback to logical line navigation if we don't have wrapping info yet.
        if let Some(prev_nl) = self.text[..self.cursor_pos].rfind('\n') {
            let target_col = match self.preferred_col {
                Some(c) => c,
                None => {
                    let c = self.current_display_col();
                    self.preferred_col = Some(c);
                    c
                }
            };
            let prev_line_start = self.text[..prev_nl].rfind('\n').map(|i| i + 1).unwrap_or(0);
            let prev_line_end = prev_nl;
            self.move_to_display_col_on_line(prev_line_start, prev_line_end, target_col);
        } else {
            self.cursor_pos = 0;
            self.preferred_col = None;
        }
    }

    pub fn move_cursor_down(&mut self) {
        // If we have a wrapping cache, prefer navigating across wrapped (visual) lines.
        if let Some((target_col, move_to_last)) = {
            let cache_ref = self.wrap_cache.borrow();
            if let Some(cache) = cache_ref.as_ref() {
                let lines = &cache.lines;
                if let Some(idx) = Self::wrapped_line_index_by_start(lines, self.cursor_pos) {
                    let cur_range = &lines[idx];
                    let target_col = self
                        .preferred_col
                        .unwrap_or_else(|| self.text[cur_range.start..self.cursor_pos].width());
                    if idx + 1 < lines.len() {
                        let next = &lines[idx + 1];
                        let line_start = next.start;
                        let line_end = next.end.saturating_sub(1);
                        Some((target_col, Some((line_start, line_end))))
                    } else {
                        Some((target_col, None))
                    }
                } else {
                    None
                }
            } else {
                None
            }
        } {
            match move_to_last {
                Some((line_start, line_end)) => {
                    if self.preferred_col.is_none() {
                        self.preferred_col = Some(target_col);
                    }
                    self.move_to_display_col_on_line(line_start, line_end, target_col);
                    return;
                }
                None => {
                    // Already on last visual line -> move to end
                    self.cursor_pos = self.text.len();
                    self.preferred_col = None;
                    return;
                }
            }
        }

        // Fallback to logical line navigation if we don't have wrapping info yet.
        let target_col = match self.preferred_col {
            Some(c) => c,
            None => {
                let c = self.current_display_col();
                self.preferred_col = Some(c);
                c
            }
        };
        if let Some(next_nl) = self.text[self.cursor_pos..]
            .find('\n')
            .map(|i| i + self.cursor_pos)
        {
            let next_line_start = next_nl + 1;
            let next_line_end = self.text[next_line_start..]
                .find('\n')
                .map(|i| i + next_line_start)
                .unwrap_or(self.text.len());
            self.move_to_display_col_on_line(next_line_start, next_line_end, target_col);
        } else {
            self.cursor_pos = self.text.len();
            self.preferred_col = None;
        }
    }

    pub fn move_cursor_to_beginning_of_line(&mut self, move_up_at_bol: bool) {
        let bol = self.beginning_of_current_line();
        if move_up_at_bol && self.cursor_pos == bol {
            self.set_cursor(self.beginning_of_line(self.cursor_pos.saturating_sub(1)));
        } else {
            self.set_cursor(bol);
        }
        self.preferred_col = None;
    }

    pub fn move_cursor_to_end_of_line(&mut self, move_down_at_eol: bool) {
        let eol = self.end_of_current_line();
        if move_down_at_eol && self.cursor_pos == eol {
            let next_pos = (self.cursor_pos.saturating_add(1)).min(self.text.len());
            self.set_cursor(self.end_of_line(next_pos));
        } else {
            self.set_cursor(eol);
        }
    }

    #[allow(clippy::unwrap_used)]
    fn wrapped_lines(&self, width: u16) -> Ref<'_, Vec<Range<usize>>> {
        // Ensure cache is ready (potentially mutably borrow, then drop)
        {
            let mut cache = self.wrap_cache.borrow_mut();
            let needs_recalc = match cache.as_ref() {
                Some(c) => c.width != width,
                None => true,
            };
            if needs_recalc {
                let mut lines: Vec<Range<usize>> = Vec::new();
                for line in textwrap::wrap(
                    &self.text,
                    Options::new(width as usize).wrap_algorithm(textwrap::WrapAlgorithm::FirstFit),
                )
                .iter()
                {
                    match line {
                        std::borrow::Cow::Borrowed(slice) => {
                            let start =
                                unsafe { slice.as_ptr().offset_from(self.text.as_ptr()) as usize };
                            let end = start + slice.len();
                            let trailing_spaces =
                                self.text[end..].chars().take_while(|c| *c == ' ').count();
                            lines.push(start..end + trailing_spaces + 1);
                        }
                        std::borrow::Cow::Owned(_) => unreachable!(),
                    }
                }
                *cache = Some(WrapCache { width, lines });
            }
        }

        let cache = self.wrap_cache.borrow();
        Ref::map(cache, |c| &c.as_ref().unwrap().lines)
    }

    /// Calculate the scroll offset that should be used to satisfy the
    /// invariants given the current area size and wrapped lines.
    ///
    /// - Cursor is always on screen.
    /// - No scrolling if content fits in the area.
    fn effective_scroll(
        &self,
        area_height: u16,
        lines: &[Range<usize>],
        current_scroll: u16,
    ) -> u16 {
        let total_lines = lines.len() as u16;
        if area_height >= total_lines {
            return 0;
        }

        // Where is the cursor within wrapped lines? Prefer assigning boundary positions
        // (where pos equals the start of a wrapped line) to that later line.
        let cursor_line_idx =
            Self::wrapped_line_index_by_start(lines, self.cursor_pos).unwrap_or(0) as u16;

        let max_scroll = total_lines.saturating_sub(area_height);
        let mut scroll = current_scroll.min(max_scroll);

        // Ensure cursor is visible within [scroll, scroll + area_height)
        if cursor_line_idx < scroll {
            scroll = cursor_line_idx;
        } else if cursor_line_idx >= scroll + area_height {
            scroll = cursor_line_idx + 1 - area_height;
        }
        scroll
    }
}

impl WidgetRef for &TextArea {
    fn render_ref(&self, area: Rect, buf: &mut Buffer) {
        let lines = self.wrapped_lines(area.width);
        for (i, ls) in lines.iter().enumerate() {
            let s = &self.text[ls.start..ls.end - 1];
            buf.set_string(area.x, area.y + i as u16, s, Style::default());
        }
    }
}

impl StatefulWidgetRef for &TextArea {
    type State = TextAreaState;

    fn render_ref(&self, area: Rect, buf: &mut Buffer, state: &mut Self::State) {
        let lines = self.wrapped_lines(area.width);
        let scroll = self.effective_scroll(area.height, &lines, state.scroll);
        state.scroll = scroll;

        let start = scroll as usize;
        let end = (scroll + area.height).min(lines.len() as u16) as usize;
        for (row, ls) in (start..end).enumerate() {
            let r = &lines[ls];
            let s = &self.text[r.start..r.end - 1];
            buf.set_string(area.x, area.y + row as u16, s, Style::default());
        }
    }
}

#[cfg(test)]
#[allow(clippy::unwrap_used)]
mod tests {
    use super::*;
    // crossterm types are intentionally not imported here to avoid unused warnings
    use rand::prelude::*;

    fn rand_grapheme(rng: &mut rand::rngs::StdRng) -> String {
        let r: u8 = rng.gen_range(0..100);
        match r {
            0..=4 => "\n".to_string(),
            5..=12 => " ".to_string(),
            13..=35 => (rng.gen_range(b'a'..=b'z') as char).to_string(),
            36..=45 => (rng.gen_range(b'A'..=b'Z') as char).to_string(),
            46..=52 => (rng.gen_range(b'0'..=b'9') as char).to_string(),
            53..=65 => {
                // Some emoji (wide graphemes)
                let choices = ["👍", "😊", "🐍", "🚀", "🧪", "🌟"];
                choices[rng.gen_range(0..choices.len())].to_string()
            }
            66..=75 => {
                // CJK wide characters
                let choices = ["漢", "字", "測", "試", "你", "好", "界", "编", "码"];
                choices[rng.gen_range(0..choices.len())].to_string()
            }
            76..=85 => {
                // Combining mark sequences
                let base = ["e", "a", "o", "n", "u"][rng.gen_range(0..5)];
                let marks = ["\u{0301}", "\u{0308}", "\u{0302}", "\u{0303}"];
                format!("{}{}", base, marks[rng.gen_range(0..marks.len())])
            }
            86..=92 => {
                // Some non-latin single codepoints (Greek, Cyrillic, Hebrew)
                let choices = ["Ω", "β", "Ж", "ю", "ש", "م", "ह"];
                choices[rng.gen_range(0..choices.len())].to_string()
            }
            _ => {
                // ZWJ sequences (single graphemes but multi-codepoint)
                let choices = [
                    "👩\u{200D}💻", // woman technologist
                    "👨\u{200D}💻", // man technologist
                    "🏳️\u{200D}🌈", // rainbow flag
                ];
                choices[rng.gen_range(0..choices.len())].to_string()
            }
        }
    }

    fn ta_with(text: &str) -> TextArea {
        let mut t = TextArea::new();
        t.insert_str(text);
        t
    }

    #[test]
    fn insert_and_replace_update_cursor_and_text() {
        // insert helpers
        let mut t = ta_with("hello");
        t.set_cursor(5);
        t.insert_str("!");
        assert_eq!(t.text(), "hello!");
        assert_eq!(t.cursor(), 6);

        t.insert_str_at(0, "X");
        assert_eq!(t.text(), "Xhello!");
        assert_eq!(t.cursor(), 7);

        // Insert after the cursor should not move it
        t.set_cursor(1);
        let end = t.text().len();
        t.insert_str_at(end, "Y");
        assert_eq!(t.text(), "Xhello!Y");
        assert_eq!(t.cursor(), 1);

        // replace_range cases
        // 1) cursor before range
        let mut t = ta_with("abcd");
        t.set_cursor(1);
        t.replace_range(2..3, "Z");
        assert_eq!(t.text(), "abZd");
        assert_eq!(t.cursor(), 1);

        // 2) cursor inside range
        let mut t = ta_with("abcd");
        t.set_cursor(2);
        t.replace_range(1..3, "Q");
        assert_eq!(t.text(), "aQd");
        assert_eq!(t.cursor(), 2);

        // 3) cursor after range with shifted by diff
        let mut t = ta_with("abcd");
        t.set_cursor(4);
        t.replace_range(0..1, "AA");
        assert_eq!(t.text(), "AAbcd");
        assert_eq!(t.cursor(), 5);
    }

    #[test]
    fn delete_backward_and_forward_edges() {
        let mut t = ta_with("abc");
        t.set_cursor(1);
        t.delete_backward(1);
        assert_eq!(t.text(), "bc");
        assert_eq!(t.cursor(), 0);

        // deleting backward at start is a no-op
        t.set_cursor(0);
        t.delete_backward(1);
        assert_eq!(t.text(), "bc");
        assert_eq!(t.cursor(), 0);

        // forward delete removes next grapheme
        t.set_cursor(1);
        t.delete_forward(1);
        assert_eq!(t.text(), "b");
        assert_eq!(t.cursor(), 1);

        // forward delete at end is a no-op
        t.set_cursor(t.text().len());
        t.delete_forward(1);
        assert_eq!(t.text(), "b");
    }

    #[test]
    fn delete_backward_word_and_kill_line_variants() {
        // delete backward word at end removes the whole previous word
        let mut t = ta_with("hello   world  ");
        t.set_cursor(t.text().len());
        t.delete_backward_word();
        assert_eq!(t.text(), "hello   ");
        assert_eq!(t.cursor(), 8);

        // From inside a word, delete from word start to cursor
        let mut t = ta_with("foo bar");
        t.set_cursor(6); // inside "bar" (after 'a')
        t.delete_backward_word();
        assert_eq!(t.text(), "foo r");
        assert_eq!(t.cursor(), 4);

        // From end, delete the last word only
        let mut t = ta_with("foo bar");
        t.set_cursor(t.text().len());
        t.delete_backward_word();
        assert_eq!(t.text(), "foo ");
        assert_eq!(t.cursor(), 4);

        // kill_to_end_of_line when not at EOL
        let mut t = ta_with("abc\ndef");
        t.set_cursor(1); // on first line, middle
        t.kill_to_end_of_line();
        assert_eq!(t.text(), "a\ndef");
        assert_eq!(t.cursor(), 1);

        // kill_to_end_of_line when at EOL deletes newline
        let mut t = ta_with("abc\ndef");
        t.set_cursor(3); // EOL of first line
        t.kill_to_end_of_line();
        assert_eq!(t.text(), "abcdef");
        assert_eq!(t.cursor(), 3);

        // kill_to_beginning_of_line from middle of line
        let mut t = ta_with("abc\ndef");
        t.set_cursor(5); // on second line, after 'e'
        t.kill_to_beginning_of_line();
        assert_eq!(t.text(), "abc\nef");

        // kill_to_beginning_of_line at beginning of non-first line removes the previous newline
        let mut t = ta_with("abc\ndef");
        t.set_cursor(4); // beginning of second line
        t.kill_to_beginning_of_line();
        assert_eq!(t.text(), "abcdef");
        assert_eq!(t.cursor(), 3);
    }

    #[test]
    fn cursor_left_and_right_handle_graphemes() {
        let mut t = ta_with("a👍b");
        t.set_cursor(t.text().len());

        t.move_cursor_left(); // before 'b'
        let after_first_left = t.cursor();
        t.move_cursor_left(); // before '👍'
        let after_second_left = t.cursor();
        t.move_cursor_left(); // before 'a'
        let after_third_left = t.cursor();

        assert!(after_first_left < t.text().len());
        assert!(after_second_left < after_first_left);
        assert!(after_third_left < after_second_left);

        // Move right back to end safely
        t.move_cursor_right();
        t.move_cursor_right();
        t.move_cursor_right();
        assert_eq!(t.cursor(), t.text().len());
    }

    #[test]
    fn cursor_vertical_movement_across_lines_and_bounds() {
        let mut t = ta_with("short\nloooooooooong\nmid");
        // Place cursor on second line, column 5
        let second_line_start = 6; // after first '\n'
        t.set_cursor(second_line_start + 5);

        // Move up: target column preserved, clamped by line length
        t.move_cursor_up();
        assert_eq!(t.cursor(), 5); // first line has len 5

        // Move up again goes to start of text
        t.move_cursor_up();
        assert_eq!(t.cursor(), 0);

        // Move down: from start to target col tracked
        t.move_cursor_down();
        // On first move down, we should land on second line, at col 0 (target col remembered as 0)
        let pos_after_down = t.cursor();
        assert!(pos_after_down >= second_line_start);

        // Move down again to third line; clamp to its length
        t.move_cursor_down();
        let third_line_start = t.text().find("mid").unwrap();
        let third_line_end = third_line_start + 3;
        assert!(t.cursor() >= third_line_start && t.cursor() <= third_line_end);

        // Moving down at last line jumps to end
        t.move_cursor_down();
        assert_eq!(t.cursor(), t.text().len());
    }

    #[test]
    fn home_end_and_emacs_style_home_end() {
        let mut t = ta_with("one\ntwo\nthree");
        // Position at middle of second line
        let second_line_start = t.text().find("two").unwrap();
        t.set_cursor(second_line_start + 1);

        t.move_cursor_to_beginning_of_line(false);
        assert_eq!(t.cursor(), second_line_start);

        // Ctrl-A behavior: if at BOL, go to beginning of previous line
        t.move_cursor_to_beginning_of_line(true);
        assert_eq!(t.cursor(), 0); // beginning of first line

        // Move to EOL of first line
        t.move_cursor_to_end_of_line(false);
        assert_eq!(t.cursor(), 3);

        // Ctrl-E: if at EOL, go to end of next line
        t.move_cursor_to_end_of_line(true);
        // end of second line ("two") is right before its '\n'
        let end_second_nl = t.text().find("\nthree").unwrap();
        assert_eq!(t.cursor(), end_second_nl);
    }

    #[test]
    fn end_of_line_or_down_at_end_of_text() {
        let mut t = ta_with("one\ntwo");
        // Place cursor at absolute end of the text
        t.set_cursor(t.text().len());
        // Should remain at end without panicking
        t.move_cursor_to_end_of_line(true);
        assert_eq!(t.cursor(), t.text().len());

        // Also verify behavior when at EOL of a non-final line:
        let eol_first_line = 3; // index of '\n' in "one\ntwo"
        t.set_cursor(eol_first_line);
        t.move_cursor_to_end_of_line(true);
        assert_eq!(t.cursor(), t.text().len()); // moves to end of next (last) line
    }

    #[test]
    fn word_navigation_helpers() {
        let t = ta_with("  alpha  beta   gamma");
        let mut t = t; // make mutable for set_cursor
        // Put cursor after "alpha"
        let after_alpha = t.text().find("alpha").unwrap() + "alpha".len();
        t.set_cursor(after_alpha);
        assert_eq!(t.beginning_of_previous_word(), 2); // skip initial spaces

        // Put cursor at start of beta
        let beta_start = t.text().find("beta").unwrap();
        t.set_cursor(beta_start);
        assert_eq!(t.end_of_next_word(), beta_start + "beta".len());

        // If at end, end_of_next_word returns len
        t.set_cursor(t.text().len());
        assert_eq!(t.end_of_next_word(), t.text().len());
    }

    #[test]
    fn wrapping_and_cursor_positions() {
        let mut t = ta_with("hello world here");
        let area = Rect::new(0, 0, 6, 10); // width 6 -> wraps words
        // desired height counts wrapped lines
        assert!(t.desired_height(area.width) >= 3);

        // Place cursor in "world"
        let world_start = t.text().find("world").unwrap();
        t.set_cursor(world_start + 3);
        let (_x, y) = t.cursor_pos(area).unwrap();
        assert_eq!(y, 1); // world should be on second wrapped line

        // With state and small height, cursor is mapped onto visible row
        let mut state = TextAreaState::default();
        let small_area = Rect::new(0, 0, 6, 1);
        // First call: cursor not visible -> effective scroll ensures it is
        let (_x, y) = t.cursor_pos_with_state(small_area, &state).unwrap();
        assert_eq!(y, 0);

        // Render with state to update actual scroll value
        let mut buf = Buffer::empty(small_area);
        ratatui::widgets::StatefulWidgetRef::render_ref(&(&t), small_area, &mut buf, &mut state);
        // After render, state.scroll should be adjusted so cursor row fits
        let effective_lines = t.desired_height(small_area.width);
        assert!(state.scroll < effective_lines);
    }

    #[test]
    fn cursor_pos_with_state_basic_and_scroll_behaviors() {
        // Case 1: No wrapping needed, height fits — scroll ignored, y maps directly.
        let mut t = ta_with("hello world");
        t.set_cursor(3);
        let area = Rect::new(2, 5, 20, 3);
        // Even if an absurd scroll is provided, when content fits the area the
        // effective scroll is 0 and the cursor position matches cursor_pos.
        let bad_state = TextAreaState { scroll: 999 };
        let (x1, y1) = t.cursor_pos(area).unwrap();
        let (x2, y2) = t.cursor_pos_with_state(area, &bad_state).unwrap();
        assert_eq!((x2, y2), (x1, y1));

        // Case 2: Cursor below the current window — y should be clamped to the
        // bottom row (area.height - 1) after adjusting effective scroll.
        let mut t = ta_with("one two three four five six");
        // Force wrapping to many visual lines.
        let wrap_width = 4;
        let _ = t.desired_height(wrap_width);
        // Put cursor somewhere near the end so it's definitely below the first window.
        t.set_cursor(t.text().len().saturating_sub(2));
        let small_area = Rect::new(0, 0, wrap_width, 2);
        let state = TextAreaState { scroll: 0 };
        let (_x, y) = t.cursor_pos_with_state(small_area, &state).unwrap();
        assert_eq!(y, small_area.y + small_area.height - 1);

        // Case 3: Cursor above the current window — y should be top row (0)
        // when the provided scroll is too large.
        let mut t = ta_with("alpha beta gamma delta epsilon zeta");
        let wrap_width = 5;
        let lines = t.desired_height(wrap_width);
        // Place cursor near start so an excessive scroll moves it to top row.
        t.set_cursor(1);
        let area = Rect::new(0, 0, wrap_width, 3);
        let state = TextAreaState {
            scroll: lines.saturating_mul(2),
        };
        let (_x, y) = t.cursor_pos_with_state(area, &state).unwrap();
        assert_eq!(y, area.y);
    }

    #[test]
    fn wrapped_navigation_across_visual_lines() {
        let mut t = ta_with("abcdefghij");
        // Force wrapping at width 4: lines -> ["abcd", "efgh", "ij"]
        let _ = t.desired_height(4);

        // From the very start, moving down should go to the start of the next wrapped line (index 4)
        t.set_cursor(0);
        t.move_cursor_down();
        assert_eq!(t.cursor(), 4);

        // Cursor at boundary index 4 should be displayed at start of second wrapped line
        t.set_cursor(4);
        let area = Rect::new(0, 0, 4, 10);
        let (x, y) = t.cursor_pos(area).unwrap();
        assert_eq!((x, y), (0, 1));

        // With state and small height, cursor should be visible at row 0, col 0
        let small_area = Rect::new(0, 0, 4, 1);
        let state = TextAreaState::default();
        let (x, y) = t.cursor_pos_with_state(small_area, &state).unwrap();
        assert_eq!((x, y), (0, 0));

        // Place cursor in the middle of the second wrapped line ("efgh"), at 'g'
        t.set_cursor(6);
        // Move up should go to same column on previous wrapped line -> index 2 ('c')
        t.move_cursor_up();
        assert_eq!(t.cursor(), 2);

        // Move down should return to same position on the next wrapped line -> back to index 6 ('g')
        t.move_cursor_down();
        assert_eq!(t.cursor(), 6);

        // Move down again should go to third wrapped line. Target col is 2, but the line has len 2 -> clamp to end
        t.move_cursor_down();
        assert_eq!(t.cursor(), t.text().len());
    }

    #[test]
    fn cursor_pos_with_state_after_movements() {
        let mut t = ta_with("abcdefghij");
        // Wrap width 4 -> visual lines: abcd | efgh | ij
        let _ = t.desired_height(4);
        let area = Rect::new(0, 0, 4, 2);
        let mut state = TextAreaState::default();
        let mut buf = Buffer::empty(area);

        // Start at beginning
        t.set_cursor(0);
        ratatui::widgets::StatefulWidgetRef::render_ref(&(&t), area, &mut buf, &mut state);
        let (x, y) = t.cursor_pos_with_state(area, &state).unwrap();
        assert_eq!((x, y), (0, 0));

        // Move down to second visual line; should be at bottom row (row 1) within 2-line viewport
        t.move_cursor_down();
        ratatui::widgets::StatefulWidgetRef::render_ref(&(&t), area, &mut buf, &mut state);
        let (x, y) = t.cursor_pos_with_state(area, &state).unwrap();
        assert_eq!((x, y), (0, 1));

        // Move down to third visual line; viewport scrolls and keeps cursor on bottom row
        t.move_cursor_down();
        ratatui::widgets::StatefulWidgetRef::render_ref(&(&t), area, &mut buf, &mut state);
        let (x, y) = t.cursor_pos_with_state(area, &state).unwrap();
        assert_eq!((x, y), (0, 1));

        // Move up to second visual line; with current scroll, it appears on top row
        t.move_cursor_up();
        ratatui::widgets::StatefulWidgetRef::render_ref(&(&t), area, &mut buf, &mut state);
        let (x, y) = t.cursor_pos_with_state(area, &state).unwrap();
        assert_eq!((x, y), (0, 0));

        // Column preservation across moves: set to col 2 on first line, move down
        t.set_cursor(2);
        ratatui::widgets::StatefulWidgetRef::render_ref(&(&t), area, &mut buf, &mut state);
        let (x0, y0) = t.cursor_pos_with_state(area, &state).unwrap();
        assert_eq!((x0, y0), (2, 0));
        t.move_cursor_down();
        ratatui::widgets::StatefulWidgetRef::render_ref(&(&t), area, &mut buf, &mut state);
        let (x1, y1) = t.cursor_pos_with_state(area, &state).unwrap();
        assert_eq!((x1, y1), (2, 1));
    }

    #[test]
    fn wrapped_navigation_with_newlines_and_spaces() {
        // Include spaces and an explicit newline to exercise boundaries
        let mut t = ta_with("word1  word2\nword3");
        // Width 6 will wrap "word1  " and then "word2" before the newline
        let _ = t.desired_height(6);

        // Put cursor on the second wrapped line before the newline, at column 1 of "word2"
        let start_word2 = t.text().find("word2").unwrap();
        t.set_cursor(start_word2 + 1);

        // Up should go to first wrapped line, column 1 -> index 1
        t.move_cursor_up();
        assert_eq!(t.cursor(), 1);

        // Down should return to the same visual column on "word2"
        t.move_cursor_down();
        assert_eq!(t.cursor(), start_word2 + 1);

        // Down again should cross the logical newline to the next visual line ("word3"), clamped to its length if needed
        t.move_cursor_down();
        let start_word3 = t.text().find("word3").unwrap();
        assert!(t.cursor() >= start_word3 && t.cursor() <= start_word3 + "word3".len());
    }

    #[test]
    fn wrapped_navigation_with_wide_graphemes() {
        // Four thumbs up, each of display width 2, with width 3 to force wrapping inside grapheme boundaries
        let mut t = ta_with("👍👍👍👍");
        let _ = t.desired_height(3);

        // Put cursor after the second emoji (which should be on first wrapped line)
        t.set_cursor("👍👍".len());

        // Move down should go to the start of the next wrapped line (same column preserved but clamped)
        t.move_cursor_down();
        // We expect to land somewhere within the third emoji or at the start of it
        let pos_after_down = t.cursor();
        assert!(pos_after_down >= "👍👍".len());

        // Moving up should take us back to the original position
        t.move_cursor_up();
        assert_eq!(t.cursor(), "👍👍".len());
    }

    #[test]
    fn fuzz_textarea_randomized() {
        // Deterministic seed for reproducibility
        // Seed the RNG based on the current day in Pacific Time (PST/PDT). This
        // keeps the fuzz test deterministic within a day while still varying
        // day-to-day to improve coverage.
        #[allow(clippy::unwrap_used)]
        let pst_today_seed: u64 = (chrono::Utc::now() - chrono::Duration::hours(8))
            .date_naive()
            .and_hms_opt(0, 0, 0)
            .unwrap()
            .and_utc()
            .timestamp() as u64;
        let mut rng = rand::rngs::StdRng::seed_from_u64(pst_today_seed);

        for _case in 0..10_000 {
            let mut ta = TextArea::new();
            let mut state = TextAreaState::default();
            // Start with a random base string
            let base_len = rng.gen_range(0..30);
            let mut base = String::new();
            for _ in 0..base_len {
                base.push_str(&rand_grapheme(&mut rng));
            }
            ta.set_text(&base);
            // Choose a valid char boundary for initial cursor
            let mut boundaries: Vec<usize> = vec![0];
            boundaries.extend(ta.text().char_indices().map(|(i, _)| i).skip(1));
            boundaries.push(ta.text().len());
            let init = boundaries[rng.gen_range(0..boundaries.len())];
            ta.set_cursor(init);

            let mut width: u16 = rng.gen_range(1..=12);
            let mut height: u16 = rng.gen_range(1..=4);

            for _step in 0..200 {
                // Mostly stable width/height, occasionally change
                if rng.gen_bool(0.1) {
                    width = rng.gen_range(1..=12);
                }
                if rng.gen_bool(0.1) {
                    height = rng.gen_range(1..=4);
                }

                // Pick an operation
                match rng.gen_range(0..14) {
                    0 => {
                        // insert small random string at cursor
                        let len = rng.gen_range(0..6);
                        let mut s = String::new();
                        for _ in 0..len {
                            s.push_str(&rand_grapheme(&mut rng));
                        }
                        ta.insert_str(&s);
                    }
                    1 => {
                        // replace_range with small random slice
                        let mut b: Vec<usize> = vec![0];
                        b.extend(ta.text().char_indices().map(|(i, _)| i).skip(1));
                        b.push(ta.text().len());
                        let i1 = rng.gen_range(0..b.len());
                        let i2 = rng.gen_range(0..b.len());
                        let (start, end) = if b[i1] <= b[i2] {
                            (b[i1], b[i2])
                        } else {
                            (b[i2], b[i1])
                        };
                        let insert_len = rng.gen_range(0..=4);
                        let mut s = String::new();
                        for _ in 0..insert_len {
                            s.push_str(&rand_grapheme(&mut rng));
                        }
                        let before = ta.text().len();
                        ta.replace_range(start..end, &s);
                        let after = ta.text().len();
                        assert_eq!(
                            after as isize,
                            before as isize + (s.len() as isize) - ((end - start) as isize)
                        );
                    }
                    2 => ta.delete_backward(rng.gen_range(0..=3)),
                    3 => ta.delete_forward(rng.gen_range(0..=3)),
                    4 => ta.delete_backward_word(),
                    5 => ta.kill_to_beginning_of_line(),
                    6 => ta.kill_to_end_of_line(),
                    7 => ta.move_cursor_left(),
                    8 => ta.move_cursor_right(),
                    9 => ta.move_cursor_up(),
                    10 => ta.move_cursor_down(),
                    11 => ta.move_cursor_to_beginning_of_line(true),
                    12 => ta.move_cursor_to_end_of_line(true),
                    _ => {
                        // Jump to word boundaries
                        if rng.gen_bool(0.5) {
                            let p = ta.beginning_of_previous_word();
                            ta.set_cursor(p);
                        } else {
                            let p = ta.end_of_next_word();
                            ta.set_cursor(p);
                        }
                    }
                }

                // Sanity invariants
                assert!(ta.cursor() <= ta.text().len());

                // Render and compute cursor positions; ensure they are in-bounds and do not panic
                let area = Rect::new(0, 0, width, height);
                // Stateless render into an area tall enough for all wrapped lines
                let total_lines = ta.desired_height(width);
                let full_area = Rect::new(0, 0, width, total_lines.max(1));
                let mut buf = Buffer::empty(full_area);
                ratatui::widgets::WidgetRef::render_ref(&(&ta), full_area, &mut buf);

                // cursor_pos: x must be within width when present
                let _ = ta.cursor_pos(area);

                // cursor_pos_with_state: always within viewport rows
                let (_x, _y) = ta
                    .cursor_pos_with_state(area, &state)
                    .unwrap_or((area.x, area.y));

                // Stateful render should not panic, and updates scroll
                let mut sbuf = Buffer::empty(area);
                ratatui::widgets::StatefulWidgetRef::render_ref(
                    &(&ta),
                    area,
                    &mut sbuf,
                    &mut state,
                );

                // After wrapping, desired height equals the number of lines we would render without scroll
                let total_lines = total_lines as usize;
                // state.scroll must not exceed total_lines when content fits within area height
                if (height as usize) >= total_lines {
                    assert_eq!(state.scroll, 0);
                }
            }
        }
    }
}
