<!-- ADD THIS SECTION RIGHT AFTER THE JOB PROGRESS CARD (around line 720) -->
<!-- RIGHT BEFORE </div></div></div></div> that closes the tab content -->

							<!-- Scraped Videos List -->
							<div v-if="customTabVideos[tab.id] && customTabVideos[tab.id].length > 0" class="card mt-3">
								<div class="card-header">
									<h6 class="mb-0">
										<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
										Scraped Videos ({{ customTabVideos[tab.id].length }})
									</h6>
								</div>
								<div class="card-body">
									<div class="table-responsive">
										<table class="table table-dark table-hover">
											<thead>
												<tr>
													<th style="width: 40%">Title</th>
													<th style="width: 40%">URL</th>
													<th style="width: 10%">Status</th>
													<th style="width: 10%">Actions</th>
												</tr>
											</thead>
											<tbody>
												<tr v-for="video in customTabVideos[tab.id].slice(0, 100)" :key="video.id">
													<td class="text-truncate" style="max-width: 300px;">{{ video.title }}</td>
													<td>
														<a :href="video.url" target="_blank" class="text-primary text-truncate d-block" style="max-width: 300px;">
															{{ video.url }}
														</a>
													</td>
													<td><span class="badge bg-success">{{ video.status }}</span></td>
													<td>
														<a :href="video.url" target="_blank" class="btn btn-sm btn-primary">
															<font-awesome-icon :icon="['fas', 'external-link-alt']" />
														</a>
													</td>
												</tr>
											</tbody>
										</table>
										<div v-if="customTabVideos[tab.id].length > 100" class="text-muted small mt-2">
											Showing first 100 of {{ customTabVideos[tab.id].length }} videos
										</div>
									</div>
								</div>
							</div>
