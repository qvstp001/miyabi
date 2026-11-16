import assert from 'node:assert/strict'
import { test } from 'vitest'

import { categoryBrowseParams } from '@/features/discover/category-params'
import { categoryFromSearch, updateCategorySearch, validateDiscoverSearch } from '@/features/discover/search'

test('combines multiple tags, resource conditions and year/month, preserving sorting and other tab pages', () => {
  const search = validateDiscoverSearch({
    view: 'category', zone: 'fc2', tagIds: ['tag-2', 'tag-1', 'tag-1'],
    year: 2026, month: 9, main: 'm,s,m', sort: 'score', order: 'asc',
    categoryPage: 5, releasedPage: 4, upcomingPage: 2
  })
  const next = updateCategorySearch(search, { tagIds: ['tag-3', 'tag-1'] })
  assert.deepEqual(categoryBrowseParams(categoryFromSearch(next)), {
    zone: 'fc2', main: ['m', 's'], tagIds: ['tag-1', 'tag-3'], year: '2026', month: '9'
  })
  assert.equal(next.categoryPage, undefined)
  assert.equal(next.releasedPage, 4)
  assert.equal(next.upcomingPage, 2)
  assert.equal(next.sort, 'score')
  assert.equal(next.order, 'asc')
})

test('clears resource conditions or tags without losing the other filters', () => {
  const search = validateDiscoverSearch({ tagIds: ['tag-1'], main: 'm,s', year: 2026 })
  assert.deepEqual(categoryBrowseParams(categoryFromSearch(updateCategorySearch(search, { main: '' }))), {
    zone: 'censored', tagIds: ['tag-1'], year: '2026'
  })
  assert.deepEqual(categoryBrowseParams(categoryFromSearch(updateCategorySearch(search, { tagIds: [] }))), {
    zone: 'censored', main: ['m', 's'], year: '2026'
  })
  // An explicit empty selection must survive validation instead of restoring a tab default.
  assert.equal(validateDiscoverSearch({ main: '' }).main, '')
})

test('changing shared resource conditions resets every affected tab page', () => {
  const search = validateDiscoverSearch({ main: 'm', categoryPage: 3, releasedPage: 4, upcomingPage: 2 })
  const next = updateCategorySearch(search, { main: 'm,s' })
  assert.equal(next.categoryPage, undefined)
  assert.equal(next.releasedPage, undefined)
  assert.equal(next.upcomingPage, undefined)
})

test('old tag and year links retain their meaning after URL validation and reload', () => {
  for (const [legacy, expected] of [
    [{ categoryID: 'year', tagID: 2026, main: 'm' }, { zone: 'censored', main: ['m'], year: '2026' }],
    [{ categoryID: 'category-1', tagID: 'tag-1' }, { zone: 'censored', tagIds: ['tag-1'] }]
  ] as const) {
    const search = validateDiscoverSearch(legacy)
    assert.deepEqual(categoryBrowseParams(categoryFromSearch(validateDiscoverSearch(search))), expected)
  }
})

test('rejects malformed combinations instead of silently querying a broader catalogue', () => {
  for (const invalid of [
    { tagIds: 'tag-1' }, { tagIds: [1] }, { tagIds: [''] }, { tagID: {} },
    { main: 'm,invalid' }, { main: ['m'] }, { year: '20' }, { month: '9' },
    { year: 2026, month: 13 }, { year: 2026, month: 0 }, { sort: 'unknown' }, { order: 'up' }
  ]) assert.throws(() => validateDiscoverSearch(invalid))
  assert.deepEqual(categoryBrowseParams(categoryFromSearch(validateDiscoverSearch({}))), { zone: 'censored' })
})
