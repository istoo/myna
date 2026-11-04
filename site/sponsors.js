// The sponsors' own links on github.com/istoo/myna's "Sponsored by" and in the
// READMEs' Sponsors row. Neither page links a sponsor directly: each goes
// to github.com/istoo/myna/go/<id>/<where>, which counts the click and sends the
// browser here (worker.js), so a sponsor's clicks can be told per place.
//
//   id     a-z, 0-9 and "-"; the same as the sponsor's partners.js id when
//          it has an in-app seat too (whose /go links are keys and site)
//   home   where the home page's card sends the browser
//   readme where the READMEs' logo sends it (README.md, README.zh-CN.md)
//
// A link stays here after its seat ends: a copy of the README or the page
// may still point at it.
export const SPONSORS = [
  { id: "yylx", home: "https://app.yylx.io/register?aff=YETONE" },
  { id: "apinebula", home: "https://apinebula.ai/ref/Gs54HwyB", readme: "https://apinebula.ai/ref/qmXqFGpr" },
  { id: "zerotoken", readme: "https://zzq.si/go/22zuh4a" },
  { id: "kimi", readme: "https://www.kimi.ai/code?aff=myna" },
  { id: "kimi-cn", readme: "https://www.kimi.com/code?aff=myna" },
  { id: "apikeyfan", readme: "https://apikey.fan/?aff=mg_readme" },
  { id: "wandapi", readme: "https://wandapi.ai/r/mynarepo" },
  { id: "fastai", home: "https://www.fastaitoken.com/register?aff=MYNA&utm=official" },
  { id: "soleapi", home: "https://soleapi.com/r/myna" },
  { id: "aigocode", home: "https://www.aigocode.net/invite/MYNA?utm_source=myna&utm_medium=home", readme: "https://www.aigocode.net/invite/MYNA?utm_source=myna&utm_medium=readme" },
];
