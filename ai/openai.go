package ai

import (
	"context"
	"fmt"
	"io"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/llms/openai"
)

func OpenAI(r io.Reader) (string, error) {
	// Initialize OpenAI LLM and embedder
	llm, err := openai.New()
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI client: %v", err)
	}

	/*embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return "", fmt.Errorf("failed to create embedder: %v", err)
	}*/

	// Initialize Chroma vector store
	/*store, err := chroma.New(chroma.WithEmbedder(embedder))
	if err != nil {
		return "", fmt.Errorf("failed to create Chroma store: %v", err)
	}*/

	// Load and process files
	/*dir := "./documents" // Directory containing your files
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return "", err
		}

		// Read file content
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}

		// Chunk the content
		chunks := chunkText(string(data), 1000)

		// Add each chunk to the vector store
		for i, chunk := range chunks {
			docID := fmt.Sprintf("%s_chunk_%d", path, i)
			err = store.Add(context.Background(), docID, chunk, nil)
			if err != nil {
				log.Printf("Failed to add chunk to store: %v", err)
			}
		}

		return "", nil
	})
	if err != nil {
		log.Fatalf("Failed to process files: %v", err)
	}*/

	outputBlock, err := documentloaders.NewText(r).Load(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to create document: %v", err)
	}

	/*_, err = store.AddDocuments(context.Background(), outputBlock)
	if err != nil {
		return "", fmt.Errorf("failed to store terminal output into Chroma store: %v", err)
	}*/

	// User query
	//query := "Explain this command line output"

	// Perform semantic search
	/*results, err := store.SimilaritySearch(context.Background(), query, 5)
	if err != nil {
		log.Fatalf("Failed to perform similarity search: %v", err)
	}*/

	// Prepare prompt with retrieved documents
	//var promptBuilder strings.Builder
	//promptBuilder.WriteString("A command line application has been executed. Can you explain its output?\n\n")
	//for _, doc := range results {
	//	promptBuilder.WriteString(fmt.Sprintf("Document: %s\n\n", doc.PageContent))
	//}
	//promptBuilder.WriteString(fmt.Sprintf("Question: %s", query))
	//promptBuilder.WriteString("Output:\n%s\n\n")

	query := "A command line application has been executed. Can you explain its output? If it is an error, you should focus more on how to fix the error rather than an explanation.\n\nOutput:\n"
	for i := range outputBlock {
		query += outputBlock[i].PageContent
	}

	// Get answer from LLM
	answer, err := llm.Call(context.Background(), query)
	if err != nil {
		return "", fmt.Errorf("failed to get response from LLM: %v", err)
	}

	return answer, nil
}
