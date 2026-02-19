// Losp — Formal Grammar (ANTLR4)
//
// losp is a streaming template language using Unicode operators instead of
// parentheses.  Nine single-character operator runes provide all structure;
// everything else is text.  Operators that open a scope require a matching
// terminator (◆).  Scopes nest arbitrarily.
//
// This grammar is the authoritative formal specification of losp syntax.
// The syntax checker (main.go) is generated from it via ANTLR4.

grammar Losp;

// ---------------------------------------------------------------------------
// Parser rules
// ---------------------------------------------------------------------------

// A losp program is zero or more top-level items followed by end-of-file.
// Top-level items may include stray terminators (◆) which the runtime
// silently ignores.  Inside operator bodies, every ◆ closes the
// innermost open scope — stray terminators cannot occur there.
program : topItem* EOF ;

topItem
    : store                              // ▼ name body ◆
    | immStore                           // ▽ name body ◆
    | execute                            // ▶ name args ◆
    | immExecute                         // ▷ name args ◆
    | deferOp                            // ◯ body ◆
    | retrieve                           // ▲ name
    | immRetrieve                        // △ name
    | placeholder                        // □ name
    | TERMINATOR                         // stray ◆ (top-level only)
    | text                               // everything else
    ;

// Body item: used inside operator scopes.  No stray TERMINATOR —
// every ◆ inside a scope closes that scope.
item
    : store                              // ▼ name body ◆
    | immStore                           // ▽ name body ◆
    | execute                            // ▶ name args ◆
    | immExecute                         // ▷ name args ◆
    | deferOp                            // ◯ body ◆
    | retrieve                           // ▲ name
    | immRetrieve                        // △ name
    | placeholder                        // □ name
    | text                               // everything else
    ;

// Operators that open a scope (require ◆ terminator).
// The body between name and ◆ is zero or more nested items.
store      : STORE      name item* TERMINATOR ;
immStore   : IMM_STORE  name item* TERMINATOR ;
execute    : EXECUTE    name item* TERMINATOR ;
immExecute : IMM_EXECUTE name item* TERMINATOR ;
deferOp    : DEFER      item* TERMINATOR ;

// Operators that take only a name (no ◆ terminator).
retrieve    : RETRIEVE     name ;
immRetrieve : IMM_RETRIEVE name ;
placeholder : PLACEHOLDER  name ;

// A name is an identifier, a builtin keyword, or a dynamically-computed
// name via a nested operator.  It is optional (?) because losp permits
// empty/dynamic names in some patterns.
name : ( IDENT | builtin | retrieve | immRetrieve | execute | immExecute )? ;

// All 33 builtin keywords.  Listed as explicit tokens so the parse tree
// distinguishes ▶SAY args ◆ from ▶MyFunc args ◆.
builtin
    : KW_TRUE    | KW_FALSE   | KW_EMPTY
    | KW_IF      | KW_COMPARE | KW_FOREACH
    | KW_SAY     | KW_READ
    | KW_COUNT   | KW_APPEND  | KW_PERSIST  | KW_LOAD
    | KW_PROMPT  | KW_EXTRACT | KW_SYSTEM
    | KW_UPPER   | KW_LOWER   | KW_TRIM
    | KW_GENERATE
    | KW_ASYNC   | KW_AWAIT   | KW_CHECK
    | KW_TIMER   | KW_TICKS   | KW_SLEEP
    | KW_CORPUS  | KW_ADD     | KW_INDEX
    | KW_SEARCH  | KW_EMBED   | KW_SIMILAR
    | KW_HISTORY | KW_RANDOM
    ;

// Plain text: one or more tokens that are not operator runes.
// Identifiers and builtin keywords are just text when they appear
// outside of a name position.
text : ( TEXT | IDENT | builtin )+ ;

// ---------------------------------------------------------------------------
// Lexer rules
// ---------------------------------------------------------------------------

// --- Operator runes (single Unicode characters) ---

STORE        : '\u25BC' ;   // ▼  Store (deferred)
IMM_STORE    : '\u25BD' ;   // ▽  Immediate store
RETRIEVE     : '\u25B2' ;   // ▲  Retrieve (deferred)
IMM_RETRIEVE : '\u25B3' ;   // △  Immediate retrieve
EXECUTE      : '\u25B6' ;   // ▶  Execute (deferred)
IMM_EXECUTE  : '\u25B7' ;   // ▷  Immediate execute
PLACEHOLDER  : '\u25A1' ;   // □  Placeholder
DEFER        : '\u25EF' ;   // ◯  Defer
TERMINATOR   : '\u25C6' ;   // ◆  Terminator

// --- Builtin keywords ---
// Listed before IDENT so that ANTLR's first-match-wins rule for
// equal-length tokens makes these match preferentially.

KW_TRUE     : 'TRUE' ;
KW_FALSE    : 'FALSE' ;
KW_EMPTY    : 'EMPTY' ;
KW_IF       : 'IF' ;
KW_COMPARE  : 'COMPARE' ;
KW_FOREACH  : 'FOREACH' ;
KW_SAY      : 'SAY' ;
KW_READ     : 'READ' ;
KW_COUNT    : 'COUNT' ;
KW_APPEND   : 'APPEND' ;
KW_PERSIST  : 'PERSIST' ;
KW_LOAD     : 'LOAD' ;
KW_PROMPT   : 'PROMPT' ;
KW_EXTRACT  : 'EXTRACT' ;
KW_SYSTEM   : 'SYSTEM' ;
KW_UPPER    : 'UPPER' ;
KW_LOWER    : 'LOWER' ;
KW_TRIM     : 'TRIM' ;
KW_GENERATE : 'GENERATE' ;
KW_ASYNC    : 'ASYNC' ;
KW_AWAIT    : 'AWAIT' ;
KW_CHECK    : 'CHECK' ;
KW_TIMER    : 'TIMER' ;
KW_TICKS    : 'TICKS' ;
KW_SLEEP    : 'SLEEP' ;
KW_CORPUS   : 'CORPUS' ;
KW_ADD      : 'ADD' ;
KW_INDEX    : 'INDEX' ;
KW_SEARCH   : 'SEARCH' ;
KW_EMBED    : 'EMBED' ;
KW_SIMILAR  : 'SIMILAR' ;
KW_HISTORY  : 'HISTORY' ;
KW_RANDOM   : 'RANDOM' ;

// --- Identifiers ---
// Unicode letters, digits, and underscore.  Longer matches beat keywords
// (e.g., "TRUEVALUE" is IDENT, not KW_TRUE + IDENT).
IDENT : [\p{L}\p{N}_]+ ;

// --- Text ---
// Any character that is not an operator rune and not an identifier character.
// This covers whitespace, punctuation, and miscellaneous symbols.
TEXT : ~[\u25BC\u25BD\u25B2\u25B3\u25B6\u25B7\u25A1\u25EF\u25C6\p{L}\p{N}_]+ ;
