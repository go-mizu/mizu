import React from "react";
import { Text, Box } from "ink";
import { marked, type Token, type Tokens } from "marked";
import { createLowlight } from "lowlight";
import { common } from "lowlight";

const lowlight = createLowlight(common);

interface Props {
  text: string;
}

export function Markdown({ text }: Props) {
  const tokens = marked.lexer(text);
  return <Box flexDirection="column">{renderTokens(tokens)}</Box>;
}

function renderTokens(tokens: Token[]): React.ReactNode[] {
  return tokens.map((token, i) => renderToken(token, i));
}

function renderToken(token: Token, key: number): React.ReactNode {
  switch (token.type) {
    case "paragraph":
      return (
        <Text key={key} wrap="wrap">
          {renderInline((token as Tokens.Paragraph).tokens || [])}
        </Text>
      );

    case "code": {
      const codeToken = token as Tokens.Code;
      const lang = codeToken.lang || "";
      const code = codeToken.text;
      return (
        <Box key={key} flexDirection="column" marginTop={0} marginBottom={0}>
          {lang && (
            <Text dimColor>{"  "}{lang}</Text>
          )}
          <Box paddingLeft={2} flexDirection="column">
            {highlightCode(code, lang)}
          </Box>
        </Box>
      );
    }

    case "heading": {
      const h = token as Tokens.Heading;
      return (
        <Text key={key} bold color="cyan">
          {renderInline(h.tokens || [])}
        </Text>
      );
    }

    case "list": {
      const list = token as Tokens.List;
      return (
        <Box key={key} flexDirection="column" paddingLeft={2}>
          {list.items.map((item, j) => (
            <Text key={j} wrap="wrap">
              <Text dimColor>{list.ordered ? `${j + 1}. ` : "• "}</Text>
              {renderInline(item.tokens || [])}
            </Text>
          ))}
        </Box>
      );
    }

    case "blockquote": {
      const bq = token as Tokens.Blockquote;
      return (
        <Box key={key} paddingLeft={1}>
          <Text dimColor>│ </Text>
          <Box flexDirection="column">
            {renderTokens(bq.tokens || [])}
          </Box>
        </Box>
      );
    }

    case "hr":
      return <Text key={key} dimColor>{"─".repeat(40)}</Text>;

    case "space":
      return null;

    default:
      if ("text" in token) {
        return <Text key={key} wrap="wrap">{(token as Tokens.Text).text}</Text>;
      }
      return null;
  }
}

function renderInline(tokens: Token[]): React.ReactNode[] {
  return tokens.map((t, i) => renderInlineToken(t, i));
}

function renderInlineToken(token: Token, key: number): React.ReactNode {
  switch (token.type) {
    case "text": {
      const txt = token as Tokens.Text;
      if (txt.tokens && txt.tokens.length > 0) {
        return <React.Fragment key={key}>{renderInline(txt.tokens)}</React.Fragment>;
      }
      return <Text key={key}>{txt.text}</Text>;
    }

    case "strong":
      return (
        <Text key={key} bold>
          {renderInline((token as Tokens.Strong).tokens || [])}
        </Text>
      );

    case "em":
      return (
        <Text key={key} italic>
          {renderInline((token as Tokens.Em).tokens || [])}
        </Text>
      );

    case "codespan":
      return (
        <Text key={key} color="yellow">
          {(token as Tokens.Codespan).text}
        </Text>
      );

    case "link":
      return (
        <Text key={key} color="cyan" underline>
          {(token as Tokens.Link).text}
        </Text>
      );

    case "br":
      return <Text key={key}>{"\n"}</Text>;

    default:
      if ("text" in token) {
        return <Text key={key}>{(token as { text: string }).text}</Text>;
      }
      return null;
  }
}

// Syntax highlighting
const HLJS_COLOR_MAP: Record<string, string> = {
  keyword: "magenta",
  built_in: "cyan",
  type: "cyan",
  literal: "blue",
  number: "yellow",
  string: "green",
  regexp: "red",
  symbol: "yellow",
  class: "cyan",
  attr: "yellow",
  function: "blue",
  title: "blue",
  params: "",
  comment: "gray",
  doctag: "gray",
  meta: "gray",
  "meta keyword": "magenta",
  "meta string": "green",
  section: "cyan",
  tag: "red",
  name: "red",
  "builtin-name": "cyan",
  attribute: "yellow",
  selector: "yellow",
  variable: "red",
  bullet: "blue",
  addition: "green",
  deletion: "red",
};

function highlightCode(code: string, lang: string): React.ReactNode {
  try {
    if (lang && lowlight.registered(lang)) {
      const tree = lowlight.highlight(lang, code);
      return renderHast(tree.children);
    }
  } catch {
    // Fallback to plain
  }
  return code.split("\n").map((line, i) => (
    <Text key={i} dimColor>{line}</Text>
  ));
}

function renderHast(nodes: any[]): React.ReactNode[] {
  return nodes.map((node, i) => {
    if (node.type === "text") {
      // Split by newlines to preserve line structure
      return <Text key={i}>{node.value}</Text>;
    }
    if (node.type === "element") {
      const className = (node.properties?.className || []).join(" ");
      const color = resolveHljsColor(className);
      const children = renderHast(node.children || []);
      if (color) {
        return <Text key={i} color={color}>{children}</Text>;
      }
      return <React.Fragment key={i}>{children}</React.Fragment>;
    }
    return null;
  });
}

function resolveHljsColor(className: string): string | undefined {
  // Classes are like "hljs-keyword", "hljs-string", etc.
  for (const cls of className.split(" ")) {
    const name = cls.replace("hljs-", "");
    if (HLJS_COLOR_MAP[name]) return HLJS_COLOR_MAP[name];
  }
  return undefined;
}
