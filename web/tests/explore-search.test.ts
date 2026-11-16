import assert from 'node:assert/strict'
import { test } from 'vitest'
import { updateExploreSearch, validateExploreSearch } from '@/features/explore/search'

test('ranking filters and the selected page survive URL validation', () => {
  const search = validateExploreSearch({ view: 'ranking', zone: 'fc2', period: 'monthly', page: '3' })
  assert.equal(search.zone, 'fc2')
  assert.equal(search.period, 'monthly')
  assert.equal(search.page, 3)
  assert.deepEqual(validateExploreSearch(search), search)
})

test('changing a content filter resets pagination while preserving unrelated choices', () => {
  const search = validateExploreSearch({ view: 'ranking', zone: 'fc2', period: 'monthly', page: 4 })
  const next = updateExploreSearch(search, { period: 'daily' })
  assert.equal(next.page, undefined)
  assert.equal(next.zone, 'fc2')
  assert.equal(next.period, 'daily')
})

test('numeric list IDs and list sorting survive URL reload', () => {
  const search = validateExploreSearch({ view: 'lists', listID: 123, sort: 'score', order: 'asc', page: 2 })
  assert.equal(search.listID, '123')
  assert.deepEqual(validateExploreSearch(search), search)
  const next = updateExploreSearch(search, { sort: 'release' })
  assert.equal(next.listID, '123')
  assert.equal(next.order, 'asc')
  assert.equal(next.page, undefined)
})

test('personal lists keep their own upstream status rather than local browse flags', () => {
  const search = validateExploreSearch({ view: 'personal', status: 'watched', page: 2 })
  const next = updateExploreSearch(search, { status: 'want_watch' })
  assert.equal(next.status, 'want_watch')
  assert.equal(next.page, undefined)
})

test('invalid feed, resource ID and sort combinations fail instead of querying another feed', () => {
  for (const invalid of [
    { view: 'other' }, { zone: 'anime' }, { zone: ['fc2'] }, { period: 'yearly' },
    { period: ['weekly'] }, { status: 'viewed' }, { listMode: 'related' },
    { listID: 'id:m:tag' }, { listID: {} }, { listID: 1.5 }, { q: [] },
    { sort: 'invalid' }, { order: 'up' }, { page: 0 }
  ]) assert.throws(() => validateExploreSearch(invalid))
})
