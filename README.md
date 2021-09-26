# About the effects of memory setting in AWS Lambda 



I have been wondering for a long time how the memory setting of a Lambda function affects the CPU configuration. I know that there is a correlation and that at the maximum memory setting (10GB) there are 6 vCPU available ( [see here](https://aws.amazon.com/about-aws/whats-new/2021/08/aws-lambda-supports-10-gb-memory-6-vcpu-cores-aws-govcloud-us-regions/?nc1=h_ls) )
 

Anecdotally, however, there should always be at least 2 vCPU available. 

Besides this question, I wanted to know if there is a sweet spot in terms of cost. Since the costs increase according to memory, but the elapsed time decreases (billing is done in ms), it could be that a certain memory setting minimizes the costs (because the duration decrease has a stronger effect than the memory increase).
I investigated these two questions empirically in the Frankfurt region using a CPU intensive routine (Mandelbrot in Golang).

## Setup
In advance, many thanks to esimov for the Mandelbrot routine (https://github.com/esimov/gobrot ). I have adopted the core of the algorithm.
The Lambda function computes a Mandelbrot image with given parameters and stores it in the /tmp folder of Lambda. Memory usage is minimal (which is desirable for this test), but CPU usage is extensive. Local testing showed that the algorithm uses the available cores to the max.
The algorithm generates as many goroutines as the image height - so 1024,2048 or 4096 for example (which I consider a lot) and uses them to calculate the image.
The code is built pretty straight forward in Go and deployed to Lambda via CLI. In the corresponding Lambda function you can then run the code using a test call and see the elapsed time and memory usage in Cloudwatch logs.

## The measurement
I tested three different image sizes (1024x1024, 2048x2048, 4096x4096) and 9 different memory settings (128/256/384/512/1024/2048/4096/8192/10240 MB). For each measurement 2x3 runs were made (each time with different memory settings so that Lambda was "cold" and „warm effects“ could be excluded). The first run was done in the afternoon of a weekday (9/18/2021), the second at night (I wanted to see if there were any differences). All tests were done in the AWS Frankfurt region (eu-central-1). So each measurement is the average of 6 individual measurements. Per measurement the cost was calculated based on the elapsed time and the memory setting.
The cost according to the AWS pricing page is $0.0000166667 for each GB-second.

## The results
The following 3 tables show the results in seconds. Cost is for 1 million computed Mandelbrot images (to make the numbers a little bit more readable). 

### Image size 1024x1024, 1024 'parallel' goroutines
| Memory/MB        | vCPU           | Duration/sec  | Cost for 1M images|
| -------------:|-------------:| -----:|-----:|
|   128|     2|     15,84|      33,00$|
|   256|     2|     8,46|      35,25$|
|   384|     2|     5,94|      37,13$|
|   512|     2|     4,65|      38,73$|
|   1024|     2|     2,85|      47,57$|
|   2048|     2|     1,94|      64,63$|
|   4096|     3|     1,51|      100,87$|
|   8192|     5|     1,31|      174,67$|
|   10240|     6|     1,29|      215,50$|


### Image size 2048x2048, 2048 'parallel' goroutines
| Memory/MB        | vCPU           | Duration/sec  | Cost for 1M images|
| -------------:|-------------:| -----:|-----:|
|   128|     2|     60,75|      126,56$|
|   256|     2|     30,55|      127,29$|
|   384|     2|     20,47|      127,94$|
|   512|     2|     15,85|      132,08$|
|   1024|     2|     8,26|      137,67$|
|   2048|     2|     4,72|      157,33$|
|   4096|     3|     3,04|      202,67$|
|   8192|     5|     2,21|      294,67$|
|   10240|    6|    2,05|      341,33$|

### Image size 4096x4096, 4096 'parallel' goroutines
| Memory/MB        | vCPU           | Duration/sec  | Cost for 1M images|
| -------------:|-------------:| -----:|-----:|
|   128|     2|     233,56|      486,58$|
|   256|     2|     117,22|      488,42$|
|   384|     2|     78,17|      488,56$|
|   512|     2|     59,06|      492,17$|
|   1024|     2|     29,49|      491,50$|
|   2048|     2|     15,34|      511,33$|
|   4096|     3|     8,92|      594,67$|
|   8192|     5|     5,73|      764,00$|
|   10240|     6|     5,21|      869,17$|





## The findings  
In general, it could be seen that the single measurements were very close (although I did not calculate the variance). Also, there was no real difference between the measurements in the afternoon (where I suspected a higher load) and at night (where I suspected that perhaps the data center was more idle and Lambda functions were taking advantage of this). 

Also validated is the statement that at least 2 vCPUs are available (determined with `runtime.NumCPU()`). However, since performance varies, these must be fractions of a vCPU; the NumCPU result is not really meaningful.

Furthermore, I don't see any cost advantages of a higher memory setting (and thus CPU setting) if you don't need it. You can see (4th column) that the sweet spot is at the lowest memory setting; that's where calculating the CPU intensive Mandelbrot image is cheapest. If you have **no** need for speed (because in batch etc) and you do need only little memory (one reason why I love Go so much - it's just damn efficient) so it's not worth configuring more memory in Lambda (hoping that the increased CPU performance will lead to a disproportionate speed advantage / cost advantage).  



## Comparison to on-premise hardware
To be able to classify the Lambda performance, I also did a test locally. I calculated a 2048x2048 image on my 2013 Core i3-4010U CPU @ 1.70GHz (8GB RAM) and my Macbook Air M1 (also 8GB RAM). 
The Lambda comparison value for this is **2.21 sec** (2048x2048 - 8GB RAM).

* Duration Core i3 / 8192MB/ 4 cores / Ubuntu: **3.52** sec
* Duration M1 / 8192MB / 8 cores / macos 11.6: **1.78** sec

Even though this comparison should be taken with a grain of salt, it is interesting to see that an i3 notebook processor from 2013(!) still holds its own in the comparison. I would have expected the performance of Lambda (especially in this very high memory configuration) to be much better. 
However, I expected that an M1 will beat everyone and everything in this comparison. I admit, I love my M1 macbook :-)



Thanks for reading, hope it was helpful. If something is missing, wrong, unclear or I made a mistake in thinking, please write me. 

