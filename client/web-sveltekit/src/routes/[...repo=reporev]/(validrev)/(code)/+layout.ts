import { dirname } from 'path'

import { catchError } from 'rxjs/operators'

import { asError, type ErrorLike } from '$lib/common'
import { fetchTreeEntries } from '$lib/loader/repo'
import { fetchRepoCommits } from '$lib/repo/api/commits'
import { requestGraphQL } from '$lib/web'

import type { LayoutLoad } from './$types'

export const load: LayoutLoad = async ({ parent, params }) => {
    const { resolvedRevision, repoName, revision } = await parent()
    return {
        deferred: {
            // Fetches the most recent commits for current blob, tree or repo root
            codeCommits: fetchRepoCommits({
                repoID: resolvedRevision.repo.id,
                revision: resolvedRevision.commitID,
                filePath: params.path ?? null,
            }),
            treeEntries: fetchTreeEntries({
                repoName,
                commitID: resolvedRevision.commitID,
                revision: revision ?? '',
                filePath: params.path ? dirname(params.path) : '.',
                first: 2500,
                requestGraphQL: options => requestGraphQL(options.request, options.variables),
            })
                .pipe(catchError((error): [ErrorLike] => [asError(error)]))
                .toPromise(),
        },
    }
}
