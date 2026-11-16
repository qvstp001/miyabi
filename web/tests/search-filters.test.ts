import assert from 'node:assert/strict'
import { test } from 'vitest'
import { validateSearchParams } from '@/features/search/search'
import { validateMetadataSearch } from '@/features/discover/metadata-search'
import { metadataBrowseParams } from '@/features/discover/metadata-params'

test('search filters survive URL validation and pagination without changing the keyword', () => {
  const search = validateSearchParams({ q: '  keyword  ', zone: 'fc2', sort: 'score', filter: 'subtitle', page: '3' })
  assert.deepEqual(search, { q: 'keyword', zone: 'fc2', sort: 'score', filter: 'subtitle', page: 3 })
  assert.deepEqual(validateSearchParams({ ...search, page: 4 }), { ...search, page: 4 })
  assert.deepEqual(validateSearchParams({ zone: 'western', filter: 'magnets' }), {
    q: undefined, zone: 'western', filter: 'magnets', sort: undefined, page: undefined
  })
})

test('search rejects unsupported sort and resource values', () => {
  for (const invalid of [{ zone: 'unknown' }, { sort: 'update' }, { filter: 'm' }, { filter: ['magnets'] }, { q: [] }, { page: 0 }]) {
    assert.throws(() => validateSearchParams(invalid))
  }
})

test('entity filmographies combine resource conditions and sorting without sending forbidden category filters', () => {
  const search = validateMetadataSearch({ kind: 'actor', id: 'actor-1', name: 'Actor', main: 's,m,m', sort: 'watched_count', order: 'asc', zone: 'fc2', page: 3 })
  assert.equal(search.sort, 'watched_count')
  assert.equal(search.order, 'asc')
  assert.equal(search.page, 3)
  assert.deepEqual(metadataBrowseParams(search), { entityType: 'actor', entityID: 'actor-1', main: ['m', 's'] })
  assert.equal(Object.hasOwn(search, 'zone'), false)
})
