/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const dupes = arrayOf(new DynamicModel({ diff: "", total: 0 }))
  app.db().newQuery(
    "SELECT diff, COUNT(*) AS total FROM news WHERE diff != '' GROUP BY diff HAVING COUNT(*) > 1",
  ).all(dupes)
  if (dupes.length > 0) {
    throw new Error(
      "news.diff must be unique before adding idx_news_diff_unique; duplicated values: " +
        dupes.map((d) => `${d.diff} (${d.total} records)`).join(", "),
    )
  }
  const collection = app.findCollectionByNameOrId("pbc_987692768")
  unmarshal({
    indexes: [
      "CREATE INDEX `idx_yET5gKEhSe` ON `news` (`canonified_title`)",
      "CREATE UNIQUE INDEX `idx_news_diff_unique` ON `news` (`diff`) WHERE `diff` != ''",
    ],
  }, collection)
  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_987692768")
  unmarshal({ indexes: ["CREATE INDEX `idx_yET5gKEhSe` ON `news` (`canonified_title`)"] }, collection)
  return app.save(collection)
})
