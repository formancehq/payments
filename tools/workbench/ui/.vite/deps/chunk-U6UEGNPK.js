import {
  markup
} from "./chunk-W2Y2KR2L.js";

// node_modules/refractor/lang/xml-doc.js
xmlDoc.displayName = "xml-doc";
xmlDoc.aliases = [];
function xmlDoc(Prism) {
  Prism.register(markup);
  (function(Prism2) {
    function insertDocComment(lang, docComment) {
      if (Prism2.languages[lang]) {
        Prism2.languages.insertBefore(lang, "comment", {
          "doc-comment": docComment
        });
      }
    }
    var tag = Prism2.languages.markup.tag;
    var slashDocComment = {
      pattern: /\/\/\/.*/,
      greedy: true,
      alias: "comment",
      inside: {
        tag
      }
    };
    var tickDocComment = {
      pattern: /'''.*/,
      greedy: true,
      alias: "comment",
      inside: {
        tag
      }
    };
    insertDocComment("csharp", slashDocComment);
    insertDocComment("fsharp", slashDocComment);
    insertDocComment("vbnet", tickDocComment);
  })(Prism);
}

export {
  xmlDoc
};
//# sourceMappingURL=chunk-U6UEGNPK.js.map
