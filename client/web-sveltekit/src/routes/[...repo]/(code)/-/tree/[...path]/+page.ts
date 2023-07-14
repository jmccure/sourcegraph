import { catchError } from 'rxjs/operators'

import { asError, isErrorLike, type ErrorLike } from '$lib/common'
import { fetchTreeEntries } from '$lib/loader/repo'
import { fetchRepoCommits, queryRepositoryComparisonFileDiffs } from '$lib/repo/api/commits'
import { requestGraphQL } from '$lib/web'

import type { PageLoad } from './$types'

export const load: PageLoad = async ({ params, parent, url }) => {
    const revisionToCompare = url.searchParams.get('rev')
    // TODO: Improve handling of resolved revision across all routes
    const { resolvedRevision: resolvedRevisionOrError, revision, repoName } = await parent()
    const resolvedRevision = isErrorLike(resolvedRevisionOrError) ? null : resolvedRevisionOrError

    return {
        deferred: {
            treeEntries: resolvedRevision
                ? fetchTreeEntries({
                      repoName,
                      commitID: resolvedRevision.commitID,
                      revision: revision ?? '',
                      filePath: params.path,
                      first: 2500,
                      requestGraphQL: options => requestGraphQL(options.request, options.variables),
                  })
                      .pipe(catchError((error): [ErrorLike] => [asError(error)]))
                      .toPromise()
                : null,
            compare:
                resolvedRevision && revisionToCompare
                    ? {
                          revisionToCompare,
                          diff: fetchRepoCommits({
                              repoID: resolvedRevision.repo.id,
                              revision: revisionToCompare,
                              filePath: params.path,
                              first: 1,
                              pageInfo: { hasNextPage: true, endCursor: '1' },
                          }).then(history =>
                              queryRepositoryComparisonFileDiffs({
                                  repo: resolvedRevision.repo.id,
                                  base: history.nodes[0]?.oid ?? null,
                                  head: revisionToCompare,
                                  paths: [params.path],
                                  first: null,
                                  after: null,
                              }).toPromise()
                          ),
                      }
                    : null,
        },
    }
}
