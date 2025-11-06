1. Same queue design as other worker thread based stuff

- Input queue, Output queue exists


## Sample workflow

0. Dedupe chunks before starting the workflow
1. Queue all chunks to input queue
2. Pull chunk from input queue from goroutine and enqueue them to Downloader
3. Pull chunk from downloader Output queue
    1. If download suceeded, continue
    2. Else, enqueue to decompressor input
4. Same as 3. Enqueue to verifier (chunks)
5. Same as 3. Enqueue to Assembler 
6. Same as 3. Enqueue to Verifier (files)
7. Put file to Output Queue if redownload not needed.
8. A goroutine subscribed to output queue that moves files to correct position in game dir from staging dir.

## Notes

- Chunks can have multiple destinations.
- Initially chunks will get enqueued with all possible destinations.
- When chunks download retry is needed in steps 3 to 4. reenqueue chunks with all destinations enabled.
- When download retry is needed in step 6-7, reenqueue chunks with destinations set to the corresponding file.