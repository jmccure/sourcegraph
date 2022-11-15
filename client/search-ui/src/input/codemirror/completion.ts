import { basename } from 'path'

import {
    autocompletion,
    startCompletion,
    completionKeymap,
    CompletionResult,
    Completion,
    snippet,
    CompletionSource,
    acceptCompletion,
    selectedCompletion,
    currentCompletions,
    setSelectedCompletion,
} from '@codemirror/autocomplete'
import { Extension, Prec } from '@codemirror/state'
import { keymap, EditorView, KeyBinding } from '@codemirror/view'
import {
    mdiCodeArray,
    mdiCodeBraces,
    mdiCodeNotEqual,
    mdiCodeString,
    mdiCube,
    mdiCubeOutline,
    mdiDrawingBox,
    mdiFileDocument,
    mdiFilterOutline,
    mdiFunction,
    mdiHistory,
    mdiKey,
    mdiLink,
    mdiMatrix,
    mdiNull,
    mdiNumeric,
    mdiPackage,
    mdiPiBox,
    mdiPillar,
    mdiPound,
    mdiShape,
    mdiSitemap,
    mdiSourceBranch,
    mdiTextBox,
    mdiTimetable,
    mdiWeb,
    mdiWrench,
} from '@mdi/js'
import * as H from 'history'
import { isEqual, startCase } from 'lodash'

import { isDefined } from '@sourcegraph/common'
import { SymbolKind } from '@sourcegraph/search'
import {
    createFilterSuggestions,
    PREDICATE_REGEX,
    regexInsertText,
    repositoryInsertText,
} from '@sourcegraph/shared/src/search/query/completion-utils'
import { decorate, DecoratedToken, toDecoration } from '@sourcegraph/shared/src/search/query/decoratedToken'
import { FILTERS, FilterType, resolveFilter } from '@sourcegraph/shared/src/search/query/filters'
import { getSuggestionQuery } from '@sourcegraph/shared/src/search/query/providers-utils'
import { scanSearchQuery } from '@sourcegraph/shared/src/search/query/scanner'
import { Filter, Token } from '@sourcegraph/shared/src/search/query/token'
import { SearchMatch } from '@sourcegraph/shared/src/search/stream'
import { createSVGIcon } from '@sourcegraph/shared/src/util/dom'
import { toPrettyBlobURL } from '@sourcegraph/shared/src/util/url'

import { queryTokens } from './parsedQuery'

type CompletionType = SymbolKind | 'queryfilter' | 'repository' | 'searchhistory'

// See SymbolIcon
const typeIconMap: Record<CompletionType, string> = {
    FILE: mdiFileDocument,
    MODULE: mdiCodeBraces,
    NAMESPACE: mdiWeb,
    PACKAGE: mdiPackage,
    CLASS: mdiSitemap,
    METHOD: mdiCubeOutline,
    PROPERTY: mdiWrench,
    FIELD: mdiTextBox,
    CONSTRUCTOR: mdiCubeOutline,
    ENUM: mdiNumeric,
    INTERFACE: mdiLink,
    FUNCTION: mdiFunction,
    VARIABLE: mdiCube,
    CONSTANT: mdiPiBox,
    STRING: mdiCodeString,
    NUMBER: mdiPound,
    BOOLEAN: mdiMatrix,
    ARRAY: mdiCodeArray,
    OBJECT: mdiDrawingBox,
    KEY: mdiKey,
    NULL: mdiNull,
    ENUMMEMBER: mdiNumeric,
    STRUCT: mdiPillar,
    EVENT: mdiTimetable,
    OPERATOR: mdiCodeNotEqual,
    TYPEPARAMETER: mdiCube,
    UNKNOWN: mdiShape,
    queryfilter: mdiFilterOutline,
    repository: mdiSourceBranch,
    searchhistory: mdiHistory,
}

interface SuggestionContext {
    position: number
    onAbort: (listener: () => void) => void
}

/**
 * A suggestion source is given a completion context, the current tokens in the
 * query and the token at the current cursor position. It returns the
 * corresponding completion results.
 * The return type is generic so that it can be used to create different
 * suggestion structures.
 */
type SuggestionSource<R, C extends SuggestionContext> = (
    context: C,
    tokens: Token[],
    tokenAtPosition?: Token
) => R | null | Promise<R | null>

export type StandardSuggestionSource = SuggestionSource<CompletionResult | null, SuggestionContext>

/**
 * searchQueryAutocompletion registers extensions for automcompletion, using the
 * provided suggestion sources.
 */
export function searchQueryAutocompletion(
    sources: StandardSuggestionSource[],
    history: H.History,
    onSubmit?: () => void,
    // By default we do not enable suggestion selection with enter because that
    // interferes with the query submission logic.
    applyOnEnter = false
): Extension {
    const override: CompletionSource[] = sources.map(source => context => {
        const position = context.pos
        const query = context.state.facet(queryTokens)
        const token = query.tokens.find(token => isTokenInRange(token, position))
        return source(
            { position, onAbort: listener => context.addEventListener('abort', listener) },
            query.tokens,
            token
        )
    })

    // Customizing how completion items are rendered
    const addToOptions: NonNullable<Parameters<typeof autocompletion>[0]>['addToOptions'] = [
        // This renders the completion icon
        {
            render(completion) {
                const icon = createSVGIcon(
                    completion.type && completion.type in typeIconMap
                        ? typeIconMap[completion.type as CompletionType]
                        : typeIconMap[SymbolKind.UNKNOWN],
                    completion.type && completion.type in typeIconMap ? completion.type : ''
                )
                return icon
            },
            // Per CodeMirror documentation, 20 is the default icon
            // position
            position: 20,
        },
        {
            render: voiceOverComma,
            // after icon
            position: 21,
        },
        {
            render: voiceOverComma,
            // after label
            position: 51,
        },
        {
            render(completion) {
                if (completion.type !== 'searchhistory') {
                    return null
                }
                const tokens = scanSearchQuery(completion.label)
                if (tokens.type !== 'success') {
                    throw new Error('this should not happen')
                }
                const nodes = tokens.term
                    .flatMap(token => decorate(token))
                    .map(token => {
                        const decoration = toDecoration(completion.label, token)
                        const node = document.createElement('span')
                        node.className = decoration.className
                        node.textContent = decoration.value
                        return node
                    })

                const container = document.createElement('div')
                container.style.whiteSpace = 'initial'
                for (const node of nodes) {
                    container.append(node)
                }
                return container
            },
            position: 30,
        },
    ]

    const completionKeymapWithTabInsteadOfEnter: KeyBinding[] = completionKeymap.map(keybinding => {
        const { run } = keybinding
        return keybinding.key === 'Enter'
            ? {
                  ...keybinding,
                  key: 'Tab',
                  run: run
                      ? (view: EditorView) => {
                            if (selectedCompletion(view.state) === null) {
                                if (currentCompletions(view.state).length > 0) {
                                    view.dispatch({ effects: setSelectedCompletion(0) })
                                    acceptCompletion(view)
                                    return true
                                }
                                return false
                            }
                            return run(view)
                        }
                      : undefined,
              }
            : keybinding
    })

    // TODO: allow users to unselect the completion by pressing UpArrow when the
    // top completion is suggested. Current behavior is that is cycles to the
    // bottom.

    return [
        // Uses the default keymapping but changes accepting suggestions from Enter
        // to Tab
        Prec.highest(
            keymap.of(
                applyOnEnter
                    ? [
                          ...completionKeymapWithTabInsteadOfEnter,
                          {
                              key: 'Enter',
                              run(view) {
                                  const selected = selectedCompletion(view.state)
                                  //   // Select first completion item if none is selected
                                  //   // and items are available.
                                  //   if (selectedCompletion(view.state) === null) {
                                  //       if (currentCompletions(view.state).length > 0) {
                                  //           view.dispatch({ effects: setSelectedCompletion(0) })
                                  //           acceptCompletion(view)
                                  //           return true
                                  //       }
                                  //       return false
                                  //   }
                                  // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-explicit-any
                                  const url = (selected as any)?.url
                                  if (typeof url === 'string') {
                                      // TODO: trigger search if the query has other tokens
                                      history.push(url)
                                  }
                                  // Otherwise apply the selected completion item
                                  // TODO: actually trigger the search
                                  const hasUserPressedDownArrow = selectedCompletion(view.state) !== null
                                  if (hasUserPressedDownArrow) {
                                      view.dispatch({ effects: setSelectedCompletion(0) })
                                      const hasAccepted = acceptCompletion(view)
                                      if (!hasAccepted) {
                                          onSubmit?.()
                                      }
                                      return hasAccepted
                                  }
                                  return false
                              },
                          },
                      ]
                    : completionKeymapWithTabInsteadOfEnter
            )
        ),
        EditorView.theme({
            '.completion-type-queryfilter > .cm-completionLabel': {
                fontWeight: 'bold',
            },
            '.cm-tooltip-autocomplete svg': {
                width: '1rem',
                height: '1rem',
                display: 'inline-block',
                boxSizing: 'content-box',
                textAlign: 'center',
                paddingRight: '0.5rem',
            },
            '.cm-tooltip-autocomplete svg path': {
                fillOpacity: 0.6,
            },
            '.completion-type-searchhistory > .cm-completionLabel': {
                display: 'none',
            },
            'li.completion-type-searchhistory': {
                height: 'initial !important',
                minHeight: '1.3rem',
            },
        }),
        EditorView.updateListener.of(update => {
            // If a filter was completed, show the completion list again for
            // filter values.
            if (update.transactions.some(transaction => transaction.isUserEvent('input.complete'))) {
                const query = update.state.facet(queryTokens)
                const token = query.tokens.find(token => isTokenInRange(token, update.state.selection.main.anchor - 1))
                if (token) {
                    startCompletion(update.view)
                }
            }
        }),
        autocompletion({
            // We define our own keymap above
            defaultKeymap: false,
            override,
            optionClass: completionItem => 'completion-type-' + (completionItem.type ?? ''),
            icons: false,
            closeOnBlur: true,
            selectOnOpen: !applyOnEnter,
            addToOptions,
        }),
    ]
}

export interface DefaultSuggestionSourcesOptions {
    fetchSuggestions: (query: string, onAbort: (listener: () => void) => void) => Promise<SearchMatch[]>
    isSourcegraphDotCom: boolean
    globbing: boolean
    disableFilterCompletion?: true
    disableSymbolCompletion?: true
    showWhenEmpty?: boolean
}

/**
 * Creates default suggestion sources to complete available filters, dynamic
 * suggestions for the current pattern and static and dynamic suggestions for
 * the current filter value.
 */
export function createDefaultSuggestionSources(
    options: DefaultSuggestionSourcesOptions
): SuggestionSource<CompletionResult | null, SuggestionContext>[] {
    const sources: SuggestionSource<CompletionResult | null, SuggestionContext>[] = []

    if (options.disableFilterCompletion !== true) {
        sources.push(
            // Static suggestions shown if the current position is outside a
            // filter value
            createDefaultSource((context, tokens, token) => {
                if (tokens.length === 0) {
                    return null
                }

                // Default to the current cursor position (e.g. if the token is a
                // whitespace, we want the suggestion to be inserted after it)
                let from = context.position

                if (token?.type === 'pattern') {
                    // If the token is a pattern (e.g. the start of a filter name),
                    // we want the suggestion to complete that name.
                    from = token.range.start
                }

                return {
                    from,
                    options: FILTER_SUGGESTIONS,
                    validFor: /^-?[a-z]+$/i,
                }
            }),
            // Show static filter value suggestions
            createFilterSource((_context, _tokens, token, resolvedFilter) => {
                if (!resolvedFilter?.definition.discreteValues) {
                    return null
                }

                const { value } = token
                const insidePredicate = value ? PREDICATE_REGEX.test(value.value) : false
                const hasDynamicSuggestions = resolvedFilter.definition.suggestions

                // Don't show static suggestions if we are inside a predicate or
                // if the filter already has a value _and_ is configured for
                // dynamic suggestions.
                // That's because dynamic suggestions are not filtered (filter: false)
                // which CodeMirror always displays above filtered suggestions.
                if (insidePredicate || (value && hasDynamicSuggestions)) {
                    return null
                }

                return {
                    from: value?.range.start ?? token.range.end,
                    to: value?.range.end,
                    // Filtering is unnecessary when dynamic suggestions are
                    // available because if there is any input that the static
                    // suggestions could be filtered by we disable static
                    // suggestions and only show the dynamic ones anyway.
                    filter: !hasDynamicSuggestions,
                    options: resolvedFilter.definition
                        .discreteValues(value, options.isSourcegraphDotCom)
                        .map(({ label, insertText, asSnippet }, index) => {
                            const apply = (insertText || label) + ' '
                            return {
                                label,
                                // See issue https://github.com/sourcegraph/sourcegraph/issues/38254
                                // Per CodeMirror's documentation (https://codemirror.net/docs/ref/#autocomplete.snippet)
                                // "The user can move between fields with Tab and Shift-Tab as long as the fields are
                                // active. Moving to the last field or moving the cursor out of the current field
                                // deactivates the fields."
                                // This means we need to append a field at the end so that pressing Tab when at the last
                                // field will move the cursor after the filter value and not move focus outside the input
                                apply: asSnippet ? snippet(apply + '${}') : apply,
                                // Setting boost this way has the effect of
                                // displaying matching suggestions in the same
                                // order as they have been defined in code.
                                boost: index * -1,
                            }
                        }),
                    // validFor should not be set when the filter has dynamic
                    // suggestions, otherwise static suggestions won't be
                    // removed from the list (because we also disable
                    // filtering above)
                    validFor: hasDynamicSuggestions ? undefined : /^[().:a-z]+$/i,
                }
            }),

            // Show dynamic filter value suggestions
            createFilterSource(async (context, tokens, token, resolvedFilter) => {
                // On Sourcegraph.com, prompt only static suggestions (above) if there is no value to use for generating dynamic suggestions yet.
                if (
                    options.isSourcegraphDotCom &&
                    (!token.value || (token.value.type === 'literal' && token.value.value === ''))
                ) {
                    return null
                }

                if (!resolvedFilter?.definition.suggestions) {
                    return null
                }

                const results = await options.fetchSuggestions(
                    getSuggestionQuery(tokens, token, resolvedFilter.definition.suggestions),
                    context.onAbort
                )
                if (results.length === 0) {
                    return null
                }
                const filteredResults = results
                    .filter(match => match.type === resolvedFilter.definition.suggestions)
                    .flatMap(match => completionFromSearchMatch(match, options, token, tokens, token.value?.value))
                    .filter(isDefined)

                const insidePredicate = token.value ? PREDICATE_REGEX.test(token.value.value) : false

                return {
                    from: token.value?.range.start ?? token.range.end,
                    to: token.value?.range.end,
                    filter: false,
                    options: filteredResults,
                    getMatch: insidePredicate || options.globbing ? undefined : createMatchFunction(token),
                }
            })
        )
    }

    if (options.disableSymbolCompletion !== true) {
        sources.push(
            // Show symbol suggestions outside of filters
            createDefaultSource(async (context, tokens, token) => {
                if (!token || token.type !== 'pattern') {
                    return null
                }

                const results = await options.fetchSuggestions(
                    getSuggestionQuery(tokens, token, suggestionTypeFromTokens(tokens)),
                    context.onAbort
                )
                if (results.length === 0) {
                    return null
                }

                return {
                    from: token.range.start,
                    to: token.range.end,
                    options: results
                        .flatMap(match => completionFromSearchMatch(match, options, token, tokens))
                        .filter(isDefined),
                }
            })
        )
    }

    return sources
}

// Returns what kind of type to query for based on existing tokens in the query
function suggestionTypeFromTokens(tokens: Token[]): SearchMatch['type'] {
    let isWithinRepo = false
    let isWithinFile = false
    for (const token of tokens) {
        if (token.type !== 'filter') {
            continue
        }
        switch (token.field.value) {
            case 'type':
                switch (token.value?.value) {
                    case 'symbol':
                        return 'symbol'
                    case 'path':
                    case 'file':
                        return 'path'
                    case 'repo':
                        return 'repo'
                    case 'diff':
                    case 'commit':
                        return 'commit'
                }
                break
            case 'r':
            case 'repo':
                isWithinRepo = true
                break
            case 'file':
            case 'path':
                isWithinFile = true
                break
        }
    }
    if (isWithinRepo && isWithinFile) {
        return 'symbol'
    }
    if (isWithinRepo) {
        return 'path'
    }
    return 'repo'
}

interface CompletionWithURL extends Completion {
    url?: string
}

function completionFromSearchMatch(
    match: SearchMatch,
    options: DefaultSuggestionSourcesOptions,
    activeToken: Token,
    tokens: Token[],
    filterValue?: string
): CompletionWithURL[] {
    const hasNonActivePatternTokens =
        tokens.find(token => token.type === 'pattern' && !isEqual(token.range, activeToken.range)) !== undefined
    const isDefaultSource = filterValue === undefined
    switch (match.type) {
        case 'path':
            return [
                {
                    label: match.path,
                    type: SymbolKind.FILE,
                    url: hasNonActivePatternTokens
                        ? undefined
                        : toPrettyBlobURL({
                              filePath: match.path,
                              revision: match.commit,
                              repoName: match.repository,
                          }),
                    apply: (isDefaultSource ? 'file:' : '') + regexInsertText(match.path, options) + ' ',
                    info: match.repository,
                },
            ]
        case 'repo':
            return [
                {
                    label: match.repository,
                    type: 'repository',
                    url: hasNonActivePatternTokens ? undefined : `/${match.repository}`,
                    apply:
                        (isDefaultSource ? 'repo:' : '') +
                        repositoryInsertText(match, { ...options, filterValue }) +
                        ' ',
                },
            ]
        case 'symbol':
            return match.symbols.map(symbol => ({
                label: symbol.name,
                type: symbol.kind,
                url: hasNonActivePatternTokens ? undefined : symbol.url,
                apply: symbol.name + ' ',
                detail: `${startCase(symbol.kind.toLowerCase())} | ${basename(match.path)}`,
                info: match.repository,
            }))
        default:
            return []
    }
}

/**
 * Creates a suggestion source that triggers on no token or pattern or whitespace
 * tokens.
 */
function createDefaultSource<R, C extends SuggestionContext>(source: SuggestionSource<R, C>): SuggestionSource<R, C> {
    return (context, tokens, token) => {
        if (token && token.type !== 'pattern' && token.type !== 'whitespace') {
            return null
        }
        return source(context, tokens, token)
    }
}

type FilterSuggestionSource<R, C extends SuggestionContext> = (
    context: C,
    tokens: Token[],
    filter: Filter,
    resolvedFilter: ReturnType<typeof resolveFilter>
) => ReturnType<SuggestionSource<R, C>>

/**
 * Creates a suggestion source that triggers when a filter value is completed.
 */
function createFilterSource<R, C extends SuggestionContext>(
    source: FilterSuggestionSource<R, C>
): SuggestionSource<R, C> {
    return (context, tokens, token) => {
        // Not completing filter value
        if (!token || token.type !== 'filter' || (token.value && token.value.range.start > context.position)) {
            return null
        }

        const resolvedFilter = resolveFilter(token.field.value)
        if (!resolvedFilter) {
            return null
        }

        return source(context, tokens, token, resolvedFilter)
    }
}

const FILTER_SUGGESTIONS: Completion[] = createFilterSuggestions(Object.keys(FILTERS) as FilterType[]).map(
    ({ label, insertText, detail }) => ({
        label,
        type: 'queryfilter',
        apply: insertText,
        detail,
        boost: insertText.startsWith('-') ? 1 : 2, // demote negated filters
    })
)

/**
 * This helper function creates a function suitable for CodeMirror's 'getMatch'
 * option. This is used to allow CodeMirror to highlight the matching part of
 * the label.
 * See https://codemirror.net/docs/ref/#autocomplete.CompletionResult.getMatch
 */
function createMatchFunction(token: Filter): ((completion: Completion) => number[]) | undefined {
    if (!token.value?.value) {
        return undefined
    }
    try {
        // Creating a regular expression fails if the value contains special
        // regex characters in invalid positions. In that case we don't
        // highlight.
        const pattern = new RegExp(token.value.value, 'ig')
        return completion => Array.from(completion.label.matchAll(pattern), matchToIndexTuple).flat()
    } catch {
        return undefined
    }
}

/**
 * Converts a regular expression match into an (possibly empty) number tuple
 * representing the start index and the end index of the match.
 */
function matchToIndexTuple(match: RegExpMatchArray): number[] {
    return match.index !== undefined ? [match.index, match.index + match[0].length] : []
}

// Looks like there might be a bug with how the end range for a field is
// computed? Need to add 1 to make this work properly.
function isTokenInRange(
    token: { type: DecoratedToken['type']; range: { start: number; end: number } },
    position: number
): boolean {
    return token.range.start <= position && token.range.end + (token.type === 'field' ? 2 : 0) >= position
}

/**
 * Helper function for inserting voice over-only commas to improve the way the
 * content is read.
 */
function voiceOverComma(): HTMLElement {
    const element = document.createElement('span')
    element.className = 'sr-only'
    element.append(document.createTextNode(','))
    return element
}
