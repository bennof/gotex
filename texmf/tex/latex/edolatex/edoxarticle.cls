\NeedsTeXFormat{LaTeX2e}
\ProvidesClass{edoxarticle}[2025/09/28 EDOX article class]

\LoadClass{bfarticle}
\RequirePackage{edoxbase}
\automark{section}

%\newcounter{exercise}[subsection]
%\newcounter{subexercise}[exercise]

\renewcommand*\theexercise{\thesubsection.\arabic{exercise}}