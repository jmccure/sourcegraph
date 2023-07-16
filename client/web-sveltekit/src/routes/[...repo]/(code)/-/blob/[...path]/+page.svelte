<script lang="ts">
    import { mdiFileCodeOutline } from '@mdi/js'

    import { parseQueryAndHash } from '@sourcegraph/shared/src/util/url'

    import { goto } from '$app/navigation'
    import { page } from '$app/stores'
    import CodeMirrorBlob from '$lib/CodeMirrorBlob.svelte'
    import { addLineRangeQueryParameter, formatSearchParameters, toPositionOrRangeQueryParameter } from '$lib/common'
    import type { BlobFileFields } from '$lib/graphql-operations'
    import Icon from '$lib/Icon.svelte'
    import FileHeader from '$lib/repo/FileHeader.svelte'
    import { asStore } from '$lib/utils'
    import type { SelectedLineRange } from '$lib/web'

    import type { PageData } from './$types'
    import FormatAction from './FormatAction.svelte'
    import WrapLinesAction, { lineWrap } from './WrapLinesAction.svelte'

    export let data: PageData

    $: blob = asStore(data.blob.deferred)
    $: highlights = asStore(data.highlights.deferred)
    $: loading = $blob.loading
    let blobData: BlobFileFields
    $: if (!$blob.loading && $blob.data) {
        blobData = $blob.data
    }
    $: formatted = !!blobData?.richHTML
    $: showRaw = $page.url.searchParams.get('view') === 'raw'
    $: selectedPosition = parseQueryAndHash($page.url.search, $page.url.hash)

    function updateURLWithLineInformation(event: { detail: SelectedLineRange }) {
        const range = event.detail
        const parameters = new URLSearchParams($page.url.searchParams)
        parameters.delete('popover')

        let query: string | undefined

        if (range?.line !== range?.endLine && range?.endLine) {
            query = toPositionOrRangeQueryParameter({
                range: {
                    start: { line: range.line },
                    end: { line: range.endLine },
                },
            })
        } else if (range?.line) {
            query = toPositionOrRangeQueryParameter({ position: { line: range.line } })
        }

        const newSearchParameters = formatSearchParameters(addLineRangeQueryParameter(parameters, query))
        goto('?' + newSearchParameters)
    }
</script>

<FileHeader>
    <Icon slot="icon" svgPath={mdiFileCodeOutline} />
    <svelte:fragment slot="actions">
        {#if !formatted || showRaw}
            <WrapLinesAction />
        {/if}
        {#if formatted}
            <FormatAction />
        {/if}
    </svelte:fragment>
</FileHeader>

<div class="content" class:loading>
    {#if blobData}
        {#if blobData.richHTML && !showRaw}
            <div class="rich">
                {@html blobData.richHTML}
            </div>
        {:else}
            <CodeMirrorBlob
                blob={blobData}
                highlights={($highlights && !$highlights.loading && $highlights.data) || ''}
                wrapLines={$lineWrap}
                selectedLines={selectedPosition.line ? selectedPosition : null}
                on:selectline={updateURLWithLineInformation}
            />
        {/if}
    {/if}
</div>

<style lang="scss">
    .content {
        display: flex;
        overflow-x: auto;
        flex: 1;
    }
    .loading {
        filter: blur(1px);
    }

    .rich {
        padding: 1rem;
        overflow: auto;
    }
</style>
