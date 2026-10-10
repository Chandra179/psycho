# What does your writing say about you? A free, private, open method to find out

**Suggested title:** Psycho: A Private, Transparent Psycholinguistic Analysis You Can Run in Your Browser

**Alternative titles:**
- Your Words, Your Device: Language-Based Personality Analysis That Never Leaves Your Browser
- I Built a Writing Analyzer That Shows Its Math

---

## The idea

The words we choose carry signals about how we think. Researchers have studied this for decades. James Pennebaker and Laura King showed in 1999 that small, everyday words (pronouns, articles, negations) relate to stable individual differences. Tal Yarkoni's 2010 study, *Personality in 100,000 words*, mapped which word categories go with each of the Big Five personality traits across blog authors.

I turned that research into a tool: **Psycho**, a psycholinguistic analysis you can try at **https://psycho.chan179.com**.

## How it works

1. **Give it your writing.** Paste text, or load a .txt or .md file. A journal entry, a blog post or an essay all work. Around 500 words or more gives the most meaningful results.
2. **It counts word patterns.** Each word is matched against a dictionary of categories (pronouns, emotion words, cognitive words, and so on), following the dictionary approach behind LIWC.
3. **It scores nine measures.** The Big Five traits use the published word-category correlations from Yarkoni (2010). Cognitive Style uses the function-word index from Pennebaker et al. (2014). Regulatory Focus, Need for Cognition, Need for Closure and Schwartz values are language-based estimates built from the psychology literature.
4. **It ranks the result.** Each score is placed against a reference set of 3,992 blog posts, so you can see where your text sits compared with typical writing.

## What makes it different

**Private by design.** The analysis runs entirely in your browser, using code compiled to WebAssembly. Your text is never uploaded, and there is no account and no server that sees it. If you want to reopen a reading later, you can choose to keep it on your own device, and delete it any time.

**Transparent calculation.** Most personality tools give you a score and ask you to trust it. Psycho shows its work. Open the *Calculation details* section of the report and you can see every step: which word categories matched, how many of your words fell into each, the weight each category carries, and how the final score and ranking were reached. Matching words appear in context, so you can check them against your own text.

**Export what you need.** Save the report as a single HTML file or as a PDF.

## An honest note on accuracy

I want to be clear about what this is. In a test on 2,442 essays, the scores matched people's questionnaire answers only slightly better than a coin flip (AUC between 0.53 and 0.56, where 0.5 is chance). Word counting cannot read sarcasm or context, and a sample of 500 words is a thin slice of a person.

So Psycho is for curiosity and self-reflection, not for hiring, clinical or other decisions about a person. Each report says this up front, shows a range around every score, and says "too close to call" when a score sits on a boundary. I published the weak results alongside the strong ones because a tool that shows its limits is more useful than one that hides them.

## Try it

Paste something you have written and open the calculation details. If you find a word category that looks wrong, or a result that seems off, I would like to hear about it.

**https://psycho.chan179.com**

*Source code and method notes: https://github.com/Chandra179/psycho*

---

### References

- Pennebaker, J.W., & King, L.A. (1999). Linguistic styles: Language use as an individual difference. *Journal of Personality and Social Psychology*.
- Yarkoni, T. (2010). Personality in 100,000 words. *Journal of Research in Personality*.
- Pennebaker, J.W., Chung, C.K., Frazee, J., Lavergne, G.M., & Beaver, D.I. (2014). When small words foretell academic success: The case of college admissions essays. *PLoS ONE*.
- Pennebaker, J.W., Boyd, R.L., Jordan, K., & Blackburn, K. (2015). The development and psychometric properties of LIWC2015.
- Mairesse, F., Walker, M.A., Mehl, M.R., & Moore, R.K. (2007). Using linguistic cues for the automatic recognition of personality in conversation and text. *Journal of Artificial Intelligence Research*.

---

**Suggested post caption (with the video):**
Your words say more than you think. I built a free tool that reads the word patterns in your writing, backed by published psycholinguistics research. It runs 100% in your browser, so nothing is uploaded, and every score comes with its full calculation. Export to HTML or PDF. Try it: psycho.chan179.com
